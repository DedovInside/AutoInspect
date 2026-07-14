/**
 * Domain: user profile (USER / ADMIN roles).
 *
 * Primary path: GET/PATCH `/v1/auth/me` + normalization.
 */

import { apiClient } from "./apiClient";
import { normalizeApiError } from "./apiFoundation";
import { normalizeUser, updateStoredUser } from "./authService";
import {
  normalizeUserProfile,
  userProfileToApiBody,
} from "./userProfileNormalize";

export {
  normalizeUserProfile,
  userProfileToApiBody,
  resolveProfileDisplayName,
} from "./userProfileNormalize";

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

function throwProfileLoadError(err) {
  const n = normalizeApiError(err);
  if (n.code === "aborted") {
    throw err instanceof Error ? err : new Error(n.message || "Запрос отменён");
  }
  throw new Error(
    typeof n.message === "string" && n.message.trim()
      ? n.message
      : "Не удалось загрузить профиль"
  );
}

function throwProfileSaveError(err) {
  const n = normalizeApiError(err);

  if (n.code === "aborted") {
    throw err instanceof Error ? err : new Error(n.message || "Запрос отменён");
  }

  if (n.status === 404 || n.status === 405) {
    throw new Error("Сохранение профиля пока недоступно на сервере");
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

/**
 * GET /v1/auth/me
 * @param {{ signal?: AbortSignal, timeoutMs?: number }} [options]
 */
export async function getMyUserProfile(options = {}) {
  const { signal: userSignal, timeoutMs = 20_000 } = options ?? {};
  const { signal: timeoutSig, clear } = abortAfter(timeoutMs);
  const combined = combineAbortSignals(userSignal, timeoutSig);

  try {
    /** @type {unknown} */
    const raw = await apiClient.get("/v1/auth/me", {
      signal: combined,
      auth: true,
    });
    return normalizeUserProfile(raw ?? {});
  } catch (err) {
    throwProfileLoadError(err);
  } finally {
    clear();
  }
}

/**
 * PATCH /v1/auth/me
 * @param {{ contact_name?: string, contact_phone?: string, contact_email?: string }} profile
 * @param {{ signal?: AbortSignal, timeoutMs?: number }} [options]
 */
export async function saveMyUserProfile(profile, options = {}) {
  const { signal: userSignal, timeoutMs = 35_000 } = options ?? {};
  const { signal: timeoutSig, clear } = abortAfter(timeoutMs);
  const combined = combineAbortSignals(userSignal, timeoutSig);

  try {
    /** @type {unknown} */
    const raw = await apiClient.patch(
      "/v1/auth/me",
      userProfileToApiBody(profile),
      { signal: combined, auth: true }
    );

    const normalized = normalizeUserProfile(raw ?? profile);
    const authUser = normalizeUser(normalized);
    if (authUser) {
      updateStoredUser(authUser);
    }
    return normalized;
  } catch (err) {
    throwProfileSaveError(err);
  } finally {
    clear();
  }
}
