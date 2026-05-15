import {
  buildApiUrl,
  devLogApiRoundTrip,
  previewForLog,
  resolveApiBaseUrl,
  safeJsonParse,
  sanitizePayloadForLog,
} from "./apiFoundation";
import { authDevLog } from "./authDebug";
import { normalizeAuthResponse, normalizeUser } from "./authNormalize";

const ACCESS_TOKEN_KEY = "access_token";
const REFRESH_TOKEN_KEY = "refresh_token";
const USER_KEY = "auth_user";

/** Bumped on {@link clearSession} to discard in-flight refresh / race with logout */
let authEpoch = 0;

/** Single-flight refresh shared with apiClient 401 retry. */
let refreshInFlight = null;

export function isDevAuthBypassEnabled() {
  return import.meta.env.DEV && import.meta.env.VITE_DEV_AUTH_BYPASS === "true";
}

/**
 * Recoverable outages: keep tokens + last good user snapshot when possible.
 * @param {unknown} error
 */
function shouldPreserveSessionDuringOutage(error) {
  const status =
    error && typeof error === "object" && "status" in error
      ? /** @type {{ status?: number }} */ (error).status
      : undefined;

  const code =
    error && typeof error === "object" && "code" in error
      ? /** @type {{ code?: string }} */ (error).code
      : undefined;

  if (status === 0 || code === "network_error") return true;
  /** Treat server errors / gateway as downtime rather than wiping auth instantly */
  if (typeof status === "number" && status >= 500) return true;
  return false;
}

/**
 * Reads response body safely; 204 / empty → null.
 * @param {Response} response
 */
async function readAuthResponseBody(response) {
  if (response.status === 204 || response.status === 205) {
    return null;
  }
  const raw = await response.text();
  if (!raw.trim()) return null;
  const parsed = safeJsonParse(raw);
  if (parsed.ok) return parsed.value;
  /** Non-JSON body */
  return { _nonJsonBody: parsed.raw };
}

function buildJsonBody(value) {
  if (typeof value === "string") return value;
  return JSON.stringify(value ?? {});
}

/**
 * Authenticated bootstrap calls use raw fetch (multipart not used).
 * Mirrors apiClient base URL behaviour without importing apiClient (no cycles).
 *
 * @param {string} path
 * @param {RequestInit & { bodyObj?: unknown }} options
 */
async function request(path, options = {}) {
  const { bodyObj, headers: hdrsInput, ...rest } = /** @type {any} */ (options);
  const url = buildApiUrl(resolveApiBaseUrl(), path);
  const method = (rest.method || "GET").toUpperCase();
  const logPath = path;

  /** @type {Record<string,string>} */
  const headerObj =
    hdrsInput && typeof hdrsInput === "object" ? { ...hdrsInput } : {};

  const body =
    rest.body !== undefined
      ? rest.body
      : bodyObj !== undefined
        ? buildJsonBody(bodyObj)
        : undefined;

  const hasEntityBody =
    body !== undefined && body !== "" && !["GET", "HEAD"].includes(method);
  if (hasEntityBody && !headerObj["Content-Type"]) {
    headerObj["Content-Type"] = "application/json";
  }

  const t0 =
    typeof performance !== "undefined" && performance.now
      ? performance.now()
      : Date.now();

  let response;
  try {
    response = await fetch(url, {
      ...rest,
      method,
      headers: headerObj,
      body:
        body === undefined && ["GET", "HEAD"].includes(method)
          ? undefined
          : body,
    });
  } catch (err) {
    const elapsed =
      (typeof performance !== "undefined" && performance.now
        ? performance.now()
        : Date.now()) - t0;

    devLogApiRoundTrip({
      method,
      path: logPath,
      status: "network/error",
      durationMs: elapsed,
      requestPayload: sanitizePayloadForLog(bodyObj ?? body),
      responseBodyPreview:
        err?.name === "AbortError" ? "[aborted]" : previewForLog(String(err)),
      note: err?.name === "AbortError" ? undefined : "auth fetch rejected",
    });

    if (err?.name === "AbortError") {
      const aborted = /** @type {Error} */ (err);
      const error = /** @type {Error & { status?: number; code?: string }} */ (
        new Error(aborted.message || "Запрос отменён")
      );
      error.status = 0;
      error.code = "aborted";
      throw error;
    }

    const error = /** @type {Error & { status?: number; code?: string }} */ (
      new Error("Не удалось связаться с сервером")
    );
    error.status = 0;
    error.code = "network_error";
    throw error;
  }

  const payload = await readAuthResponseBody(response);
  const elapsed =
    (typeof performance !== "undefined" && performance.now
      ? performance.now()
      : Date.now()) - t0;

  devLogApiRoundTrip({
    method,
    path: logPath,
    status: response.status,
    durationMs: elapsed,
    requestPayload: sanitizePayloadForLog(bodyObj ?? body),
    responseBodyPreview: previewForLog(payload),
  });

  if (!response.ok) {
    const p =
      payload &&
      typeof payload === "object" &&
      !Array.isArray(payload) &&
      !(/** @type {{ _nonJsonBody?: unknown }} */ (payload))._nonJsonBody
        ? payload
        : null;
    const fromNonJson =
      payload &&
      typeof payload === "object" &&
      (/** @type {{ _nonJsonBody?: string }} */ (payload))._nonJsonBody;

    const message =
      (p && p.message) ||
      (fromNonJson ? `HTTP ${response.status}` : `HTTP ${response.status}`);
    const error = /** @type {Error & { status?: number; code?: string; details?: unknown }} */ (
      new Error(message)
    );
    error.status = response.status;
    error.code = p?.code;
    error.details =
      fromNonJson
        ? { bodySnippet: String(fromNonJson).slice(0, 800) }
        : p ?? undefined;
    throw error;
  }

  /** Success path: reject unexpected non-JSON for auth JSON APIs */
  if (
    payload &&
    typeof payload === "object" &&
    (/** @type {{ _nonJsonBody?: string }} */ (payload))._nonJsonBody
  ) {
    throw new Error("Некорректный JSON-ответ от сервера");
  }

  return payload;
}

/**
 * Atomic best-effort persistence: all keys or none (user removed on token failure).
 * @param {{ access_token: string, refresh_token: string }} tokens
 * @param {ReturnType<typeof normalizeUser>} user
 */
function persistSession(tokens, user) {
  if (!user) {
    throw new Error("Некорректный пользователь");
  }
  try {
    localStorage.setItem(ACCESS_TOKEN_KEY, tokens.access_token);
    localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refresh_token);
    localStorage.setItem(USER_KEY, JSON.stringify(user));
  } catch (e) {
    authDevLog("storage.write_failed", {
      message: e instanceof Error ? e.message : String(e),
    });
    clearSession();
    throw e instanceof Error ? e : new Error("Ошибка записи сессии");
  }
}

/**
 * Apply OAuth/refresh tokens + user. Invalid payload → full clear, no partial state.
 * @param {unknown} authResponse
 * @param {{ source?: string }} [meta]
 */
function applyAuthResult(authResponse, meta = {}) {
  const normalized = normalizeAuthResponse(authResponse);
  if (!normalized) {
    authDevLog("applyAuthResult.invalid_payload", {
      source: meta.source ?? "unknown",
    });
    clearSession();
    throw new Error("Ответ авторизации недействителен");
  }
  persistSession(normalized.tokens, normalized.user);
  authDevLog("applyAuthResult.ok", { source: meta.source ?? "unknown" });
}

export function getAccessToken() {
  return localStorage.getItem(ACCESS_TOKEN_KEY);
}

export function getRefreshToken() {
  return localStorage.getItem(REFRESH_TOKEN_KEY);
}

export function getStoredUser() {
  const raw = localStorage.getItem(USER_KEY);
  if (!raw) {
    return null;
  }

  try {
    const parsed = JSON.parse(raw);
    const user = normalizeUser(parsed);
    if (!user) {
      localStorage.removeItem(USER_KEY);
    }
    return user;
  } catch {
    localStorage.removeItem(USER_KEY);
    return null;
  }
}

export function clearSession() {
  authEpoch += 1;
  localStorage.removeItem(ACCESS_TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
  localStorage.removeItem(USER_KEY);
  authDevLog("session.cleared", { epoch: authEpoch });
}

export function hasAccessToken() {
  return isDevAuthBypassEnabled() || Boolean(getAccessToken());
}

/**
 * @returns {boolean} true if session was cleared while this generation was in flight
 */
function isAuthGenerationStale(since) {
  return since !== authEpoch;
}

export async function startYandexOAuth() {
  authDevLog("oauth.start");
  const payload = await request("/v1/auth/yandex/start", {
    method: "GET",
  });

  const url =
    payload &&
    typeof payload === "object" &&
    ("auth_url" in payload || "authUrl" in payload)
      ? /** @type {{ auth_url?: unknown; authUrl?: unknown }} */ (payload)
          .auth_url ??
        /** @type {{ authUrl?: unknown }} */ (payload).authUrl
      : null;

  if (typeof url !== "string" || !url.trim()) {
    authDevLog("oauth.start.invalid_response", { hasPayload: Boolean(payload) });
    throw new Error("Сервер не вернул URL для входа");
  }

  return url.trim();
}

export async function exchangeYandexCode(code, state) {
  authDevLog("oauth.callback.exchange", { hasCode: Boolean(code), hasState: Boolean(state) });
  const authResponse = await request("/v1/auth/oauth/yandex", {
    method: "POST",
    bodyObj: { code, state },
  });

  applyAuthResult(authResponse, { source: "oauth_exchange" });
  return authResponse;
}

async function refreshSessionOnce() {
  const epochSnap = authEpoch;
  const refreshToken = getRefreshToken();
  if (!refreshToken) {
    throw new Error("Нет refresh token");
  }

  authDevLog("refresh.start", {});

  try {
    const authResponse = await request("/v1/auth/refresh", {
      method: "POST",
      bodyObj: { refresh_token: refreshToken },
    });

    if (isAuthGenerationStale(epochSnap)) {
      authDevLog("refresh.discarded", { reason: "session_revoked" });
      const err = /** @type {Error & { code?: string }} */ (
        new Error("Сессия сброшена")
      );
      err.code = "session_revoked";
      throw err;
    }

    applyAuthResult(authResponse, { source: "refresh" });
    authDevLog("refresh.success", {});
    return authResponse;
  } catch (e) {
    if (isAuthGenerationStale(epochSnap)) {
      authDevLog("refresh.aborted", { reason: "session_revoked" });
      throw e;
    }
    authDevLog("refresh.failed", {
      status: /** @type {{ status?: number }} */ (e)?.status,
      code: /** @type {{ code?: string }} */ (e)?.code,
    });
    clearSession();
    throw e;
  }
}

/** Shared single-flight refresh (apiClient 401 retry + ensureSession). */
export async function refreshSession() {
  const refreshToken = getRefreshToken();
  if (!refreshToken) {
    throw new Error("Нет refresh token");
  }

  if (!refreshInFlight) {
    refreshInFlight = refreshSessionOnce().finally(() => {
      refreshInFlight = null;
    });
  }

  return refreshInFlight;
}

export async function getMe() {
  const accessToken = getAccessToken();
  if (!accessToken) {
    throw new Error("Нет access token");
  }

  const payload = await request("/v1/auth/me", {
    method: "GET",
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  });

  const user = normalizeUser(payload ?? {});
  if (!user) {
    authDevLog("getMe.invalid_user_payload", {});
    clearSession();
    const err = /** @type {Error & { status?: number; code?: string }} */ (
      new Error("Не удалось прочитать профиль пользователя")
    );
    err.status = 422;
    err.code = "invalid_user_payload";
    throw err;
  }

  try {
    localStorage.setItem(USER_KEY, JSON.stringify(user));
  } catch (e) {
    authDevLog("getMe.storage_failed", {});
    clearSession();
    throw e instanceof Error ? e : new Error("Ошибка записи профиля");
  }

  return user;
}

/**
 * Hydrate session after reload. Never throws — AuthContext relies on deterministic completion.
 */
export async function ensureSession() {
  if (!getAccessToken()) {
    authDevLog("ensureSession.skip", { reason: "no_access_token" });
    return null;
  }

  try {
    authDevLog("ensureSession.bootstrap", {});
    return await getMe();
  } catch (error) {
    if ((/** @type {{ status?: number }} */ (error))?.status === 401) {
      authDevLog("ensureSession.refresh_attempt", {});
      try {
        await refreshSession();
      } catch {
        authDevLog("ensureSession.after_refresh.failed", {});
        return null;
      }
      try {
        return await getMe();
      } catch (e2) {
        if (shouldPreserveSessionDuringOutage(e2)) {
          const degraded = getStoredUser();
          if (degraded) {
            authDevLog("ensureSession.degraded.after_refresh", {});
            return degraded;
          }
        }
        return null;
      }
    }

    if (shouldPreserveSessionDuringOutage(error)) {
      const degraded = getStoredUser();
      if (degraded) {
        authDevLog("ensureSession.degraded", {});
        return degraded;
      }
    }

    if ((/** @type {{ status?: number }} */ (error))?.status >= 400) {
      authDevLog("ensureSession.hard_failure", {
        status: /** @type {{ status?: number }} */ (error)?.status,
      });
    }

    return null;
  }
}

/** Best-effort server logout — local session always terminates. */
export async function logout() {
  authDevLog("logout.start", {});
  const accessToken = getAccessToken();
  const refreshToken = getRefreshToken();

  if (accessToken && refreshToken) {
    try {
      await request("/v1/auth/logout", {
        method: "POST",
        headers: {
          Authorization: `Bearer ${accessToken}`,
        },
        bodyObj: { refresh_token: refreshToken },
      });
    } catch {
      /* ignore — local cleanup always runs */
    }
  }

  clearSession();
  authDevLog("logout.done", {});
}

export function getUserRole() {
  if (import.meta.env.DEV && import.meta.env.VITE_DEV_ROLE) {
    return import.meta.env.VITE_DEV_ROLE;
  }
  const user = getStoredUser();
  return user?.role || "USER";
}

export { normalizeAuthResponse, normalizeUser };
