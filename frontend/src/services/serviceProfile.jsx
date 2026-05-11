/**
 * Domain: autoservice profile (SERVICE role).
 *
 * Primary path: GET/PUT `/v1/services/me` + normalization.
 *
 * Predictable persistence:
 * - `VITE_SERVICE_PROFILE_LOCAL_STORAGE`:
 *    `false` → never read/write localStorage,
 *    `true`  → persist profile to localStorage (cache shim),
 *    unset  → persist only while mock mode is enabled.
 *
 * Mock mode: dev default unless `VITE_USE_MOCK_SERVICE_PROFILE=false`;
 * prod: mocks only if `VITE_USE_MOCK_SERVICE_PROFILE=true`.
 */

import { apiClient } from "./apiClient";
import { normalizeApiError } from "./apiFoundation";
import { MOCK_SERVICE_PROFILE } from "./mockData";
import { serviceProfileDevLog } from "./serviceProfileDebug";
import {
  coerceString,
  mergeServiceProfileResponse,
  normalizeServiceProfile,
  normalizeServiceProfilePayload,
} from "./serviceProfileNormalize";

export {
  mergeServiceProfileResponse,
  normalizeServiceProfile,
  normalizeServiceProfilePayload,
  normalizeServicesList,
  normalizeServiceEntry,
  normalizeLevelArray,
  unwrapServiceProfileEnvelope,
  coerceString,
} from "./serviceProfileNormalize";

const STORAGE_KEY = "service_profile";

function delay(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function shouldUseServiceProfileMocks() {
  if (import.meta.env.VITE_USE_MOCK_SERVICE_PROFILE === "true") return true;
  if (import.meta.env.VITE_USE_MOCK_SERVICE_PROFILE === "false") return false;
  return import.meta.env.DEV;
}

/**
 * @returns {"none" | "local"}
 */
function persistenceMode() {
  const forced = import.meta.env.VITE_SERVICE_PROFILE_LOCAL_STORAGE;
  if (forced === "true") return "local";
  if (forced === "false") return "none";
  return shouldUseServiceProfileMocks() ? "local" : "none";
}

/**
 * @param {AbortSignal[] | (AbortSignal | undefined)[]} signals
 */
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

/** @param {number} ms */
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

function readStoredRawObject() {
  if (typeof window === "undefined") return null;
  try {
    const s = window.localStorage.getItem(STORAGE_KEY);
    if (!s) return null;
    const parsed = JSON.parse(s);
    return typeof parsed === "object" && parsed !== null && !Array.isArray(parsed)
      ? parsed
      : null;
  } catch {
    if (import.meta.env.DEV) {
      serviceProfileDevLog("local.parse_fail", {});
    }
    return null;
  }
}

function persistLocalMerged(mergedNormalized) {
  if (persistenceMode() !== "local" || typeof window === "undefined") return;

  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(mergedNormalized));
  } catch {
    serviceProfileDevLog("local.write_fail", {});
  }
}

function mockDefaultProfileNormalized() {
  return normalizeServiceProfile(MOCK_SERVICE_PROFILE);
}

function hasMeaningfulCachedProfile(norm) {
  return coerceString(norm.name, "") !== "" || norm.services.length > 0;
}

/** Map save errors → UI-safe messages */
function throwProfileSaveError(err) {
  const n = normalizeApiError(err);

  serviceProfileDevLog("save.fail", {
    aborted: n.code === "aborted",
    status: n.status,
    code: n.code,
  });

  if (n.code === "aborted") {
    throw err instanceof Error ? err : new Error(n.message || "Запрос отменён");
  }

  if (n.status === 422 || n.status === 400) {
    throw new Error(n.message || "Проверьте заполнение полей профиля");
  }

  if (n.status === 0) {
    throw new Error(n.message || "Не удалось связаться с сервером");
  }

  throw new Error(
    typeof n.message === "string" && n.message.trim()
      ? n.message
      : "Не удалось сохранить профиль"
  );
}

function throwProfileLoadError(err) {
  const n = normalizeApiError(err);

  serviceProfileDevLog("get.fail", {
    aborted: n.code === "aborted",
    status: n.status,
    code: n.code,
  });

  if (n.code === "aborted") {
    throw err instanceof Error ? err : new Error(n.message || "Запрос отменён");
  }

  throw new Error(
    typeof n.message === "string" && n.message.trim()
      ? n.message
      : "Не удалось загрузить профиль"
  );
}

/**
 * Seed for first paint. When persistence is disabled, empty profile avoids mock leakage into API-first mode.
 */
export function readMyServiceProfileFromCache() {
  if (persistenceMode() === "none") {
    return normalizeServiceProfile({});
  }
  const parsed = readStoredRawObject();
  if (parsed) {
    if (import.meta.env.DEV) {
      serviceProfileDevLog("cache.read_seed", { hasStored: true });
    }
    return normalizeServiceProfile(parsed);
  }
  return mockDefaultProfileNormalized();
}

async function fetchRealProfileCombined(options = {}) {
  const { signal: userSignal, timeoutMs = 20_000 } = options ?? {};
  const { signal: timeoutSig, clear } = abortAfter(timeoutMs);
  const combined = combineAbortSignals(userSignal, timeoutSig);

  try {
    /** @type {unknown} */
    const raw = await apiClient.get("/v1/services/me", {
      signal: combined,
      auth: true,
    });
    return normalizeServiceProfile(raw ?? {});
  } catch (err) {
    const n = normalizeApiError(err);
    if (n.status === 404) {
      if (import.meta.env.DEV) {
        serviceProfileDevLog("get.fresh_404", {});
      }
      return normalizeServiceProfile({});
    }
    throwProfileLoadError(err);
  } finally {
    clear();
  }
}

/**
 * GET /v1/services/me
 * @param {{ signal?: AbortSignal, timeoutMs?: number }} [options]
 */
export async function getMyServiceProfile(options = {}) {
  if (shouldUseServiceProfileMocks()) {
    await delay(120);
    serviceProfileDevLog("get.start", { mock: true });

    /** @type {ReturnType<typeof normalizeServiceProfile>} */
    let out;

    if (persistenceMode() === "local") {
      const stored = readStoredRawObject();
      out = normalizeServiceProfile(stored ?? MOCK_SERVICE_PROFILE);
    } else {
      out = mockDefaultProfileNormalized();
    }

    serviceProfileDevLog("get.success", {
      mock: true,
      servicesCount: out.services.length,
    });
    return out;
  }

  serviceProfileDevLog("get.start", { mock: false });

  try {
    const merged = await fetchRealProfileCombined(options);

    persistLocalMerged(merged);

    serviceProfileDevLog("get.success", {
      mock: false,
      servicesCount: merged.services.length,
    });

    return merged;
  } catch (e) {
    const n = normalizeApiError(e);

    if (n.code !== "aborted" && persistenceMode() === "local") {
      const fallback = normalizeServiceProfile(readStoredRawObject() ?? {});
      if (hasMeaningfulCachedProfile(fallback)) {
        if (import.meta.env.DEV) {
          serviceProfileDevLog("get.network_fallback_cache", {});
        }
        return fallback;
      }
    }

    throw e;
  }
}

let saveQueued = Promise.resolve();

/**
 * @param {() => Promise<unknown>} fn
 */
function enqueueSave(fn) {
  const exec = async () => {
    await saveQueued.catch(() => {});
    return fn();
  };
  const p = exec();
  saveQueued = p.catch(() => {});
  return p;
}

/**
 * PUT /v1/services/me — serialized overlapping writes.
 *
 * @param {unknown} profile
 * @param {{ signal?: AbortSignal, timeoutMs?: number }} [options]
 */
export async function saveMyServiceProfile(profile, options = {}) {
  serviceProfileDevLog("save.start", {});

  return enqueueSave(async () => {
    const payloadNormalized = normalizeServiceProfilePayload(profile);

    if (shouldUseServiceProfileMocks()) {
      await delay(200);
      const merged = mergeServiceProfileResponse(
        payloadNormalized,
        payloadNormalized
      );
      persistLocalMerged(merged);
      serviceProfileDevLog("save.success", { mock: true });
      return merged;
    }

    const { signal: userSignal, timeoutMs = 35_000 } = options ?? {};
    const { signal: timeoutSig, clear } = abortAfter(timeoutMs);
    const combined = combineAbortSignals(userSignal, timeoutSig);

    try {
      /** @type {unknown} */
      const raw = await apiClient.put(
        "/v1/services/me",
        payloadNormalized,
        { signal: combined, auth: true }
      );

      if (
        raw === null ||
        (typeof raw === "object" &&
          !Array.isArray(raw) &&
          Object.keys(raw && typeof raw === "object" ? /** @type {object} */ (raw) : {}).length === 0)
      ) {
        if (import.meta.env.DEV) {
          serviceProfileDevLog("save.partial_body", {});
        }
      }

      const merged = mergeServiceProfileResponse(
        payloadNormalized,
        raw ?? {}
      );
      persistLocalMerged(merged);

      serviceProfileDevLog("save.success", { mock: false });
      return merged;
    } catch (err) {
      throwProfileSaveError(err);
    } finally {
      clear();
    }
  });
}
