/**
 * Minimal API foundation: URL building, safe JSON parsing, error normalization,
 * and development-only integration logging (no tokens, no multipart file bytes).
 */

const MAX_BODY_LOG_CHARS = 2000;

/**
 * Normalized trailing-slash-free base URL from env (relative "" is valid for Vite proxy).
 */
export function resolveApiBaseUrl() {
  const raw = import.meta.env.VITE_API_BASE_URL ?? "";
  return String(raw).replace(/\/+$/, "");
}

/**
 * Join base + path without double slashes. Absolute URLs pass through unchanged.
 * @param {string} base
 * @param {string} path
 */
export function buildApiUrl(base, path) {
  const p = String(path ?? "");
  if (/^https?:\/\//i.test(p)) return p;

  const b = String(base ?? "").replace(/\/+$/, "");
  const rel = p.startsWith("/") ? p : `/${p}`;
  if (!b) return rel;
  return `${b}${rel}`;
}

/** @param {unknown} value */
export function isFormData(value) {
  return typeof FormData !== "undefined" && value instanceof FormData;
}

/**
 * @param {string} text
 * @returns {{ ok: true, value: unknown } | { ok: false, raw: string }}
 */
export function safeJsonParse(text) {
  const s = String(text ?? "").trim();
  if (!s) return { ok: true, value: null };
  try {
    return { ok: true, value: JSON.parse(s) };
  } catch {
    return { ok: false, raw: s };
  }
}

/**
 * @param {unknown} body
 */
export function sanitizePayloadForLog(body) {
  if (body === undefined || body === null) return body;
  if (isFormData(body)) {
    try {
      const keys = [...new Set(body.keys())];
      let fileFields = 0;
      keys.forEach((k) => {
        const entries = body.getAll(k);
        entries.forEach((v) => {
          if (v instanceof Blob) fileFields += 1;
        });
      });
      return {
        kind: "FormData",
        fieldNames: keys,
        fileFieldCount: fileFields,
      };
    } catch {
      return { kind: "FormData", fieldNames: "[unreadable]" };
    }
  }
  if (typeof body === "string") {
    return body.length > MAX_BODY_LOG_CHARS
      ? `${body.slice(0, MAX_BODY_LOG_CHARS)}…`
      : body;
  }
  try {
    const str = JSON.stringify(body);
    return str.length > MAX_BODY_LOG_CHARS ? `${str.slice(0, MAX_BODY_LOG_CHARS)}…` : body;
  } catch {
    return "[unserializable]";
  }
}

function truncateForLog(s, max = MAX_BODY_LOG_CHARS) {
  const str = String(s ?? "");
  return str.length > max ? `${str.slice(0, max)}…` : str;
}

/**
 * Development-only structured log for integration debugging.
 * Never logs Authorization or token values.
 */
export function devLogApiRoundTrip({
  method,
  path,
  status,
  durationMs,
  requestPayload,
  responseBodyPreview,
  note,
}) {
  if (!import.meta.env.DEV) return;
  const line = {
    method: method || "?",
    path: path || "?",
    status: status ?? "?",
    durationMs: durationMs != null ? Math.round(durationMs) : "?",
    request: requestPayload,
    response: responseBodyPreview,
  };
  if (note) line.note = note;
  console.debug("[AutoInspect API]", line);
}

/**
 * Normalize any thrown/rejected value into a safe, serializable shape.
 * Does not import ApiError (avoids circular deps) — uses duck-typing.
 *
 * @param {unknown} error
 * @returns {{ status: number, message: string, code?: string, details?: unknown, raw?: unknown }}
 */
export function normalizeApiError(error) {
  const fallback = {
    status: 0,
    message: "Неизвестная ошибка",
    code: undefined,
    details: undefined,
    raw: error,
  };

  if (error == null) {
    return { ...fallback, message: "Неизвестная ошибка", raw: error };
  }

  if (typeof error === "object" && error !== null) {
    const name = /** @type {{ name?: string }} */ (error).name;

    if (name === "AbortError") {
      const msg =
        /** @type {Error} */ (error).message || "Запрос отменён";
      return {
        status: 0,
        message: msg,
        code: "aborted",
        details: undefined,
        raw: error,
      };
    }

    if (name === "ApiError") {
      const e = /** @type {{ status?: number; message?: string; code?: string; details?: unknown }} */ (
        error
      );
      return {
        status: typeof e.status === "number" ? e.status : 0,
        message: e.message || `HTTP ${e.status ?? ""}`,
        code: e.code,
        details: e.details,
        raw: error,
      };
    }
  }

  if (typeof error === "object" && error !== null && "status" in error) {
    const e = /** @type {{ status?: number; message?: string; code?: string }} */ (error);
    if (typeof e.status === "number") {
      return {
        status: e.status,
        message: e.message || `HTTP ${e.status}`,
        code: e.code,
        details: "details" in error ? /** @type {{ details?: unknown }} */ (error).details : undefined,
        raw: error,
      };
    }
  }

  if (error instanceof Error) {
    return {
      status: 0,
      message: error.message || "Ошибка",
      code: undefined,
      details: undefined,
      raw: error,
    };
  }

  try {
    return {
      status: 0,
      message: String(error),
      code: undefined,
      details: undefined,
      raw: error,
    };
  } catch {
    return fallback;
  }
}

/**
 * Build a short preview for logs (never includes huge blobs).
 * @param {unknown} parsedOrRaw
 */
export function previewForLog(parsedOrRaw) {
  if (parsedOrRaw === null || parsedOrRaw === undefined) return parsedOrRaw;
  if (typeof parsedOrRaw === "string") return truncateForLog(parsedOrRaw, 800);
  try {
    return truncateForLog(JSON.stringify(parsedOrRaw), MAX_BODY_LOG_CHARS);
  } catch {
    return "[unserializable]";
  }
}
