/**
 * Unified global error model and HTTP/retry classification.
 * Builds on apiFoundation.normalizeApiError — does not replace ApiError.
 */

import { normalizeApiError } from "./apiFoundation";
import {
  getGlobalMaxRetries,
  getRetryDelay,
  GLOBAL_MAX_RETRY,
} from "./globalApiPolicy";

/** @typedef {"auth"|"analysis"|"repair"|"admin"|"profile"|"forms"} GlobalErrorSource */

/**
 * @typedef {Object} GlobalApiErrorShape
 * @property {string} code
 * @property {number | null} status
 * @property {string} message
 * @property {GlobalErrorSource} [source]
 * @property {boolean} retryable
 * @property {unknown} [raw]
 */

export class GlobalApiError extends Error {
  /**
   * @param {GlobalApiErrorShape} p
   */
  constructor({ code, status, message, source, retryable, raw }) {
    super(message || "Ошибка запроса");
    this.name = "GlobalApiError";
    this.code = code;
    this.status = status ?? null;
    this.source = source;
    this.retryable = !!retryable;
    this.raw = raw;
  }
}

/**
 * @param {unknown} err
 * @param {GlobalErrorSource} [source]
 * @returns {GlobalApiError}
 */
export function normalizeGlobalError(err, source = "analysis") {
  const n = normalizeApiError(err);
  const retryable = isRetryableNormalized(n);
  const code =
    n.code ||
    (n.status === 0 ? "network" : n.status ? `http_${n.status}` : "unknown");

  return new GlobalApiError({
    code,
    status: typeof n.status === "number" ? n.status : null,
    message: n.message || "Неизвестная ошибка",
    source,
    retryable,
    raw: n.raw ?? err,
  });
}

/** @param {{ status: number, code?: string }} n */
function isRetryableNormalized(n) {
  if (n.code === "aborted") return false;
  const s = n.status;
  if (s === 0) return true;
  if (s === 408 || s === 425) return true;
  if (s === 429) return true;
  if (s >= 500 && s <= 599) return true;
  return false;
}

/**
 * @param {unknown} err
 */
export function isRetryableError(err) {
  if (err instanceof GlobalApiError) return err.retryable;
  return isRetryableNormalized(normalizeApiError(err));
}

/**
 * Map HTTP status + optional body hint to GlobalApiError (no fetch).
 *
 * @param {number} status
 * @param {unknown} body
 * @param {GlobalErrorSource} source
 */
export function mapHttpErrorToGlobalError(status, body, source = "analysis") {
  let message = `HTTP ${status}`;
  if (body && typeof body === "object" && !Array.isArray(body)) {
    const o = /** @type {Record<string, unknown>} */ (body);
    const m =
      (typeof o.message === "string" && o.message) ||
      (typeof o.detail === "string" && o.detail) ||
      (typeof o.error === "string" && o.error);
    if (m) message = m;
  } else if (typeof body === "string" && body.trim()) {
    message = body.trim().slice(0, 500);
  }

  const synthetic = { status, message };
  const retryable = isRetryableNormalized(
    /** @type {{ status: number, code?: string }} */ (synthetic)
  );

  return new GlobalApiError({
    code: status ? `http_${status}` : "http_error",
    status,
    message,
    source,
    retryable,
    raw: body,
  });
}

/**
 * Whether a failed request should be retried (policy + error + attempt budget).
 *
 * @param {unknown} error
 * @param {GlobalErrorSource} domain
 * @param {number} attempt 0-based
 * @param {{ isMutation?: boolean }} [ctx]
 */
export function shouldRetryRequest(error, domain, attempt, ctx = {}) {
  const max = getGlobalMaxRetries(domain);
  if (attempt >= max) return false;
  if (
    (ctx.isMutation && GLOBAL_MAX_RETRY[domain] === 0) ||
    (ctx.isMutation && domain === "admin")
  ) {
    return false;
  }
  if (!isRetryableError(error)) return false;
  const n = normalizeApiError(error);
  if (n.code === "aborted") return false;
  return true;
}

export { getRetryDelay };
