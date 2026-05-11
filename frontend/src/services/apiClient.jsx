/**
 * Shared HTTP client for AutoInspect.
 *
 * Responsibilities:
 *   - Prefix requests with VITE_API_BASE_URL (or use Vite's /v1 dev proxy).
 *   - Inject `Authorization: Bearer <access_token>` for authenticated calls.
 *   - On 401: refresh via authService.refreshSession() (single-flight mutex there)
 *     and retry the original request once.
 *   - Normalize errors into ApiError (+ normalizeApiError helper in apiFoundation).
 *   - Support JSON and multipart/form-data, AbortSignal, 204 No Content.
 *
 * Designed so domain services migrate from mocks to real endpoints without UI changes.
 */

import {
  buildApiUrl,
  devLogApiRoundTrip,
  isFormData,
  normalizeApiError,
  previewForLog,
  resolveApiBaseUrl,
  safeJsonParse,
  sanitizePayloadForLog,
} from "./apiFoundation";
import { withGlobalRequest } from "./globalRequestPolicy";
import {
  clearSession,
  getAccessToken,
  getRefreshToken,
  refreshSession,
} from "./authService";

export class ApiError extends Error {
  constructor({ status, code, message, details, raw }) {
    super(message || `HTTP ${status}`);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.details = details;
    /** Original payload/cause when useful for debugging */
    this.raw = raw;
  }

  get isNetworkError() {
    return this.status === 0;
  }

  get isAuthError() {
    return this.status === 401 || this.code === "session_expired";
  }

  /** @returns {{ status: number, message: string, code?: string, details?: unknown, raw?: unknown }} */
  toNormalized() {
    return normalizeApiError(this);
  }
}

export { normalizeApiError };

function resolvedUrl(path, query) {
  const joined = buildApiUrl(resolveApiBaseUrl(), path);
  if (!query) return joined;

  const search = new URLSearchParams();
  Object.entries(query).forEach(([key, value]) => {
    if (value === undefined || value === null || value === "") return;
    if (Array.isArray(value)) {
      value.forEach((item) => search.append(key, String(item)));
    } else {
      search.append(key, String(value));
    }
  });

  const qs = search.toString();
  return qs ? `${joined}${joined.includes("?") ? "&" : "?"}${qs}` : joined;
}

/**
 * Interpret response entity body: null for empty and 204/205; object or fallback string.
 */
async function readResponsePayload(response) {
  if (response.status === 204 || response.status === 205) {
    return null;
  }

  const raw = await response.text();
  if (!raw.trim()) return null;

  const parsed = safeJsonParse(raw);
  if (parsed.ok) return parsed.value;
  return parsed.raw;
}

function buildHeaders({ headers, body, auth }) {
  const result = new Headers(headers || {});

  if (auth) {
    const token = getAccessToken();
    if (token && !result.has("Authorization")) {
      result.set("Authorization", `Bearer ${token}`);
    }
  }

  if (body !== undefined && !isFormData(body) && !result.has("Content-Type")) {
    result.set("Content-Type", "application/json");
  }

  return result;
}

function buildRequestInit(options) {
  const init = {
    method: options.method || "GET",
    headers: buildHeaders({
      headers: options.headers,
      body: options.body,
      auth: options.auth ?? true,
    }),
    signal: options.signal,
  };

  if (options.body !== undefined) {
    init.body = isFormData(options.body)
      ? options.body
      : JSON.stringify(options.body);
  }

  return init;
}

async function fetchWithInstrumentation(path, query, options) {
  const url = resolvedUrl(path, query);
  const method = (options.method || "GET").toUpperCase();
  const logPath = resolvedUrl(path, query).replace(/^https?:\/\/[^/]+/i, "");

  const t0 =
    typeof performance !== "undefined" && performance.now
      ? performance.now()
      : Date.now();

  let response;
  try {
    response = await fetch(url, buildRequestInit(options));
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
      requestPayload: sanitizePayloadForLog(options.body),
      responseBodyPreview: err?.name === "AbortError" ? "[aborted]" : previewForLog(String(err)),
      note: err?.name === "AbortError" ? undefined : "fetch rejected",
    });

    if (err?.name === "AbortError") {
      throw new ApiError({
        status: 0,
        code: "aborted",
        message: err.message || "Запрос отменён",
        details: undefined,
        raw: err,
      });
    }

    throw new ApiError({
      status: 0,
      code: "network_error",
      message: "Не удалось связаться с сервером",
      details: { cause: err?.message },
      raw: err,
    });
  }

  const payload = await readResponsePayload(response);
  const elapsed =
    (typeof performance !== "undefined" && performance.now
      ? performance.now()
      : Date.now()) - t0;

  devLogApiRoundTrip({
    method,
    path: logPath,
    status: response.status,
    durationMs: elapsed,
    requestPayload: sanitizePayloadForLog(options.body),
    responseBodyPreview: previewForLog(payload),
  });

  return { response, payload };
}

function throwHttpError(status, payload) {
  const p =
    payload && typeof payload === "object" && !Array.isArray(payload)
      ? payload
      : null;

  throw new ApiError({
    status,
    code: p?.code,
    message: p?.message || `HTTP ${status}`,
    details: p?.details,
    raw: payload,
  });
}

export async function request(path, options = {}) {
  const auth = options.auth ?? true;
  let didRefresh = false;

  const exec = () =>
    fetchWithInstrumentation(path, options.query, options);

  let { response, payload } = await exec();

  if (response.status === 401 && auth && getRefreshToken() && !didRefresh) {
    try {
      await refreshSession();
      didRefresh = true;
      ({ response, payload } = await exec());
    } catch {
      clearSession();
      throw new ApiError({
        status: 401,
        code: "session_expired",
        message: "Сессия истекла. Войдите заново.",
        raw: undefined,
      });
    }
  }

  if (response.status === 401 && auth) {
    if (didRefresh) {
      clearSession();
    }
    throw new ApiError({
      status: 401,
      code: didRefresh ? "session_expired" : "unauthorized",
      message: didRefresh
        ? "Сессия истекла. Войдите заново."
        : "Требуется авторизация",
      details: typeof payload === "object" ? payload ?? undefined : { body: payload },
      raw: payload,
    });
  }

  if (!response.ok) {
    throwHttpError(response.status, payload);
  }

  return payload;
}

/**
 * Opt-in HTTP client with global timeout/retry/dedupe/stale policy (see globalRequestPolicy).
 * Legacy `apiClient` is unchanged; migrate calls gradually by passing a `policy` object.
 *
 * @param {string} path
 * @param {Parameters<typeof request>[1]} options
 * @param {import("./globalRequestPolicy").GlobalRequestOptions} [policy]
 */
function requestWithGlobalPolicy(path, options = {}, policy = {}) {
  return withGlobalRequest(
    (signal) => request(path, { ...options, signal }),
    {
      source: "analysis",
      ...policy,
      signal: options.signal,
    }
  );
}

export const apiClient = {
  get: (path, options) => request(path, { ...options, method: "GET" }),
  post: (path, body, options) => request(path, { ...options, method: "POST", body }),
  put: (path, body, options) => request(path, { ...options, method: "PUT", body }),
  patch: (path, body, options) => request(path, { ...options, method: "PATCH", body }),
  delete: (path, options) => request(path, { ...options, method: "DELETE" }),
};

export const apiClientWithGlobalPolicies = {
  get: (path, options = {}, policy = {}) =>
    requestWithGlobalPolicy(path, { ...options, method: "GET" }, policy),
  post: (path, body, options = {}, policy = {}) =>
    requestWithGlobalPolicy(
      path,
      { ...options, method: "POST", body },
      { isMutation: true, ...policy }
    ),
  put: (path, body, options = {}, policy = {}) =>
    requestWithGlobalPolicy(
      path,
      { ...options, method: "PUT", body },
      { isMutation: true, ...policy }
    ),
  patch: (path, body, options = {}, policy = {}) =>
    requestWithGlobalPolicy(
      path,
      { ...options, method: "PATCH", body },
      { isMutation: true, ...policy }
    ),
  delete: (path, options = {}, policy = {}) =>
    requestWithGlobalPolicy(
      path,
      { ...options, method: "DELETE" },
      { isMutation: true, ...policy }
    ),
};

/** @deprecated prefer resolveApiBaseUrl from apiFoundation */
export function getApiBaseUrl() {
  return resolveApiBaseUrl();
}
