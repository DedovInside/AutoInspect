/** Development-only auth flow tracing (never logs tokens). */

function redactDeep(value, depth = 0) {
  if (depth > 6) return "[max-depth]";
  if (value === null || value === undefined) return value;
  if (typeof value !== "object") return value;

  if (Array.isArray(value)) {
    return value.map((v) => redactDeep(v, depth + 1));
  }

  const SENSITIVE = new Set([
    "access_token",
    "refresh_token",
    "accessToken",
    "refreshToken",
    "token",
    "password",
    "authorization",
    "Authorization",
  ]);

  /** @type {Record<string, unknown>} */
  const out = {};
  for (const [k, v] of Object.entries(value)) {
    if (SENSITIVE.has(k)) {
      out[k] = "[redacted]";
    } else {
      out[k] = redactDeep(v, depth + 1);
    }
  }
  return out;
}

/**
 * @param {string} stage
 * @param {Record<string, unknown>} [details]
 */
export function authDevLog(stage, details) {
  if (!import.meta.env.DEV) return;
  const safe =
    details && typeof details === "object"
      ? redactDeep(details)
      : details ?? {};
  console.debug(`[AutoInspect Auth] ${stage}`, safe);
}
