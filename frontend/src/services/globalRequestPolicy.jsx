/**
 * Global request lifecycle: dedupe, timeout, retry, stale guards, dev logging.
 * Optional wrapper for gradual service adoption — does not replace apiClient.
 */

import { normalizeApiError } from "./apiFoundation";
import {
  GLOBAL_MAX_RETRY,
  getGlobalMaxRetries,
  getGlobalTimeoutMs,
} from "./globalApiPolicy";
import {
  getRetryDelay,
  GlobalApiError,
  normalizeGlobalError,
  shouldRetryRequest,
} from "./globalErrorPolicy";

/** @typedef {"auth"|"analysis"|"repair"|"admin"|"profile"|"forms"} GlobalRequestSource */

const inflightByKey = new Map();
/** @type {Map<string, { c: AbortController, p: Promise<unknown> }>} */
const entityMutationSlots = new Map();

/**
 * Development-only: no tokens, no raw bodies; short metadata only.
 *
 * @param {string} event
 * @param {Record<string, unknown>} [data]
 */
export function globalDevLog(event, data) {
  if (!import.meta.env.DEV) return;
  const safe = { ...data };
  if ("key" in safe && typeof safe.key === "string") {
    safe.key = safe.key.slice(0, 120);
  }
  console.debug(`[AutoInspect global] ${event}`, safe);
}

function delay(ms) {
  return new Promise((r) => setTimeout(r, ms));
}

/** @param {(AbortSignal | undefined)[]} signals */
export function mergeGlobalAbortSignals(...signals) {
  return combineAbortSignals(...signals);
}

/** @param {(AbortSignal | undefined)[]} signals */
function combineAbortSignals(...signals) {
  const valid = signals.filter((s) => s != null);
  if (valid.length === 0) return undefined;
  if (typeof AbortSignal !== "undefined" && AbortSignal.any) {
    try {
      return AbortSignal.any(valid);
    } catch {
      /* ignore */
    }
  }
  const c = new AbortController();
  const onAbort = () => c.abort();
  for (const s of valid) {
    if (s.aborted) {
      onAbort();
      break;
    }
    s.addEventListener("abort", onAbort, { once: true });
  }
  return c.signal;
}

function abortAfter(ms) {
  if (!ms || ms <= 0) {
    return { signal: undefined, clear: () => {} };
  }
  const c = new AbortController();
  const t = setTimeout(() => c.abort(), ms);
  return {
    signal: c.signal,
    clear: () => clearTimeout(t),
  };
}

/**
 * @typedef {Object} StaleSeqGuard
 * @property {() => number} bump
 * @property {() => number} current
 * @property {(n: number) => boolean} isStale
 */

/**
 * @typedef {Object} GlobalRequestOptions
 * @property {GlobalRequestSource} [source]
 * @property {string} [dedupeKey]
 * @property {boolean} [dedupe]
 * @property {number} [timeoutMs]
 * @property {AbortSignal} [signal]
 * @property {number} [maxRetries] override domain max retry count (not including first attempt)
 * @property {boolean} [isMutation] set true for POST/PATCH/DELETE-style calls to disable retry when policy requires
 * @property {StaleSeqGuard} [seqGuard]
 * @property {number} [requestSeq] seq value from guard when this request started
 * @property {boolean} [throwOnStaleSuccess]
 */

/** Sentinel: withGlobalRequest resolves to this when stale and throwOnStaleSuccess is false */
export const STALE_RESULT = Symbol("global.stale_result");

/**
 * Per-scope monotonic sequence for list/poll loaders.
 * @returns {StaleSeqGuard}
 */
export function createStaleSeqGuard() {
  let seq = 0;
  return {
    bump: () => {
      seq += 1;
      return seq;
    },
    current: () => seq,
    isStale: (n) => n !== seq,
  };
}

/**
 * Drop async result if superseded by a newer seq (polling / tabs).
 *
 * @param {StaleSeqGuard} guard
 * @param {number} requestSeq
 * @template T
 * @param {T} value
 * @returns {T | typeof STALE_RESULT}
 */
export function takeIfCurrent(guard, requestSeq, value) {
  if (guard.isStale(requestSeq)) {
    globalDevLog("stale.drop", {
      requestSeq,
      current: guard.current(),
    });
    return STALE_RESULT;
  }
  return value;
}

/**
 * One in-flight mutation per entity id within a logical domain (storm / double-submit guard).
 *
 * @param {string} domainKey e.g. "repair:bid"
 * @param {string} entityId
 * @param {(signal: AbortSignal | undefined) => Promise<unknown>} fn
 */
export async function runExclusiveEntityMutation(domainKey, entityId, fn) {
  const slot = `${domainKey}:${String(entityId)}`;
  const prev = entityMutationSlots.get(slot);
  prev?.c.abort();

  const c = new AbortController();
  const p = fn(c.signal).finally(() => {
    const cur = entityMutationSlots.get(slot);
    if (cur?.c === c) entityMutationSlots.delete(slot);
  });

  entityMutationSlots.set(slot, { c, p });
  globalDevLog("mutation.exclusive", { slot });
  return p;
}

/**
 * @template T
 * @param {(signal: AbortSignal | undefined) => Promise<T>} fn
 * @param {GlobalRequestOptions} [options]
 * @returns {Promise<T | typeof STALE_RESULT>}
 */
export async function withGlobalRequest(fn, options = {}) {
  const {
    source = "analysis",
    dedupeKey,
    dedupe = !!dedupeKey,
    timeoutMs: timeoutOverride,
    signal: userSignal,
    maxRetries: maxRetriesOverride,
    isMutation = false,
    seqGuard,
    requestSeq,
    throwOnStaleSuccess = false,
  } = options;

  const timeoutMs = getGlobalTimeoutMs(source, timeoutOverride);
  const maxConfigured = getGlobalMaxRetries(source, maxRetriesOverride);
  const attemptsTotal = Math.max(1, maxConfigured + 1);
  /** Admin mutations must not retry; reads may retry up to max. */
  const mutationNoRetryWrite =
    isMutation && (source === "admin" || GLOBAL_MAX_RETRY[source] === 0);

  const runOnce = async () => {
    const { signal: timeoutSig, clear } = abortAfter(timeoutMs);
    const merged = combineAbortSignals(userSignal, timeoutSig);
    try {
      return await fn(merged);
    } finally {
      clear();
    }
  };

  const inner = async () => {
    let lastErr;

    for (let attempt = 0; attempt < attemptsTotal; attempt++) {
      try {
        globalDevLog("request.start", {
          source,
          attempt,
          dedupeKey: dedupe ? dedupeKey : undefined,
        });

        /** @type {unknown} */
        const result = await runOnce();

        let out = result;
        if (
          seqGuard != null &&
          typeof requestSeq === "number"
        ) {
          out = takeIfCurrent(seqGuard, requestSeq, result);
          if (out === STALE_RESULT && throwOnStaleSuccess) {
            throw new GlobalApiError({
              code: "stale_response",
              status: null,
              message: "Ответ устарел",
              source,
              retryable: false,
              raw: null,
            });
          }
        }

        globalDevLog("request.success", {
          source,
          attempt,
          staleDropped: out === STALE_RESULT,
        });
        return /** @type {T | typeof STALE_RESULT} */ (out);
      } catch (e) {
        lastErr = e;
        const n = normalizeApiError(e);
        if (n.code === "aborted") {
          globalDevLog("abort", { source, attempt });
          throw normalizeGlobalError(e, source);
        }
        globalDevLog("request.fail", {
          source,
          attempt,
          status: n.status,
          code: n.code,
        });

        if (
          mutationNoRetryWrite ||
          !shouldRetryRequest(e, source, attempt, { isMutation })
        ) {
          throw normalizeGlobalError(e, source);
        }

        const wait = getRetryDelay(attempt);
        globalDevLog("retry.scheduled", { source, attempt, waitMs: wait });
        await delay(wait);
      }
    }

    throw normalizeGlobalError(lastErr, source);
  };

  /** @type {Promise<unknown>} */
  const p = inner();

  if (dedupe && dedupeKey) {
    const k = `${source}:${dedupeKey}`;
    const existing = inflightByKey.get(k);
    if (existing) {
      globalDevLog("dedupe.hit", { key: k });
      return /** @type {Promise<T | typeof STALE_RESULT>} */ (existing);
    }
    inflightByKey.set(
      k,
      p.finally(() => {
        if (inflightByKey.get(k) === p) inflightByKey.delete(k);
      })
    );
  }

  return /** @type {Promise<T | typeof STALE_RESULT>} */ (p);
}
