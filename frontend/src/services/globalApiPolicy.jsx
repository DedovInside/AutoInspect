/**
 * Global API policy: timeouts, retry budgets, mock domain map, response envelope helpers.
 * Does not import apiClient — safe to use from any service layer.
 */

import { safeJsonParse } from "./apiFoundation";

/** @typedef {"auth"|"analysis"|"repair"|"admin"|"profile"|"forms"} GlobalDomain */

/** Default request timeouts (ms) per domain. */
export const GLOBAL_TIMEOUT_MS = {
  auth: 20_000,
  analysis: 45_000,
  repair: 35_000,
  admin: 60_000,
  profile: 30_000,
  forms: 35_000,
};

/**
 * Max automatic retries for idempotent-style failures (network / 502 / 503 / 429).
 * Mutations (admin write) should use max: 0 at call site.
 */
export const GLOBAL_MAX_RETRY = {
  auth: 1,
  analysis: 2,
  repair: 2,
  /** Reads may retry once; mutations are blocked in withGlobalRequest when source===admin && isMutation. */
  admin: 1,
  profile: 1,
  forms: 1,
};

/** Base delay for exponential backoff (ms). */
export const GLOBAL_RETRY_BASE_DELAY_MS = 400;

/** Max cap for backoff (ms). */
export const GLOBAL_RETRY_MAX_DELAY_MS = 8_000;

/**
 * Env keys used to decide mock usage. `auth` uses dev bypass (no mock store).
 * `admin` is true if any admin submodule mock is on.
 * `forms` covers home-side registration + training submission mocks.
 */
const DOMAIN_MOCK_ENV = {
  auth: ["VITE_DEV_AUTH_BYPASS"],
  analysis: ["VITE_USE_MOCK_ANALYSES"],
  repair: ["VITE_USE_MOCK_REPAIR_REQUESTS"],
  profile: ["VITE_USE_MOCK_SERVICE_PROFILE"],
  forms: [
    "VITE_USE_MOCK_SERVICE_REGISTRATIONS",
    "VITE_USE_MOCK_TRAINING_REQUESTS",
  ],
  admin: [
    "VITE_USE_MOCK_ADMIN_ML_MODELS",
    "VITE_USE_MOCK_SERVICE_REGISTRATIONS",
    "VITE_USE_MOCK_TRAINING_REQUESTS",
  ],
};

/** @param {string} key */
function readEnvTriState(key) {
  const v = import.meta.env[key];
  if (v === "true") return true;
  if (v === "false") return false;
  return null;
}

/**
 * Whether mock/bypass is enabled for the domain (tri-state env; dev defaults where applicable).
 * In production, only explicit `true` flags count as "enabled".
 *
 * @param {GlobalDomain} domain
 */
export function isMockEnabled(domain) {
  const keys = DOMAIN_MOCK_ENV[domain];
  if (!keys?.length) return false;

  if (import.meta.env.PROD) {
    return keys.some((k) => readEnvTriState(k) === true);
  }

  if (domain === "auth") {
    return keys.some((k) => readEnvTriState(k) === true);
  }

  const explicitOff = keys.some((k) => readEnvTriState(k) === false);
  const explicitOn = keys.some((k) => readEnvTriState(k) === true);
  if (explicitOn) return true;
  if (keys.length > 1) {
    if (explicitOff && keys.every((k) => readEnvTriState(k) === false)) {
      return false;
    }
    return import.meta.env.DEV;
  }
  if (explicitOff) return false;
  return import.meta.env.DEV;
}

/**
 * Whether runtime should use mock data/bypass for API calls.
 * Production: never unless an env flag is explicitly `true` (isMockEnabled).
 *
 * @param {GlobalDomain} domain
 */
export function shouldUseMockData(domain) {
  return isMockEnabled(domain);
}

/**
 * @param {GlobalDomain} domain
 * @param {number} [override]
 */
export function getGlobalTimeoutMs(domain, override) {
  if (typeof override === "number" && override > 0) return override;
  return GLOBAL_TIMEOUT_MS[domain] ?? 30_000;
}

/**
 * @param {GlobalDomain} domain
 * @param {number} [overrideMax]
 */
export function getGlobalMaxRetries(domain, overrideMax) {
  if (typeof overrideMax === "number" && overrideMax >= 0) return overrideMax;
  return GLOBAL_MAX_RETRY[domain] ?? 0;
}

/**
 * @param {number} attempt 0-based
 */
export function getRetryDelay(attempt) {
  const n = Math.max(0, attempt);
  const raw = GLOBAL_RETRY_BASE_DELAY_MS * 2 ** n;
  return Math.min(raw, GLOBAL_RETRY_MAX_DELAY_MS);
}

const ENVELOPE_KEYS = [
  "data",
  "result",
  "response",
  "payload",
  "body",
  "value",
];

/**
 * Unwrap one level of common API envelopes (non-destructive for primitives).
 *
 * @param {unknown} raw
 */
export function unwrapBackendEnvelope(raw) {
  if (raw == null || typeof raw !== "object" || Array.isArray(raw)) {
    return raw;
  }
  const o = /** @type {Record<string, unknown>} */ (raw);
  for (const k of ENVELOPE_KEYS) {
    if (k in o && o[k] != null) {
      return o[k];
    }
  }
  return raw;
}

/**
 * Recursively unwrap a few nested envelope layers (bounded).
 *
 * @param {unknown} raw
 * @param {number} [depth]
 */
export function safeExtractData(raw, depth = 3) {
  let cur = raw;
  for (let i = 0; i < depth; i++) {
    const next = unwrapBackendEnvelope(cur);
    if (next === cur) break;
    cur = next;
  }
  return cur;
}

/**
 * Safe parse for string bodies; objects pass through.
 *
 * @param {unknown} body
 */
export function safeParseResponse(body) {
  if (typeof body !== "string") return body;
  const p = safeJsonParse(body);
  if (p.ok) return p.value;
  return { _unparsed: true, raw: p.raw };
}
