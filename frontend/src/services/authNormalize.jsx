/**
 * Normalize auth payloads for backend variance (camelCase, nested `data`, etc.).
 * Primary contract remains snake_case `{ tokens: { access_token, refresh_token }, user }`.
 */

import { ROLES } from "../auth/roles";

const ALLOWED = new Set(Object.values(ROLES));

/**
 * Unwrap nested API envelopes without treating arrays as payloads.
 * @param {unknown} raw
 */
function unwrapAuthBundle(raw) {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return raw;
  const withData =
    raw.data !== undefined &&
    typeof raw.data === "object" &&
    !Array.isArray(raw.data)
      ? raw.data
      : raw;

  /** @type {Record<string, unknown>} */
  const d = /** @type {Record<string, unknown>} */ (withData);
  /** Some APIs nest twice */
  if (
    d.data !== undefined &&
    typeof d.data === "object" &&
    !Array.isArray(d.data) &&
    (d.tokens ||
      (d.access_token ?? d.accessToken) ||
      (d.user !== undefined &&
        typeof d.user === "object"))
  ) {
    return d.data;
  }
  return withData;
}

/**
 * @param {unknown} tokensBlock
 */
function extractTokens(bundle) {
  if (!bundle || typeof bundle !== "object" || Array.isArray(bundle)) return null;

  /** @type {Record<string, unknown>} */
  const b = /** @type {Record<string, unknown>} */ (bundle);

  const t =
    (b.tokens && typeof b.tokens === "object"
      ? b.tokens
      : b.token ??
        b.credentials ??
        null) ?? null;

  /** @type {string | undefined} */
  let access_token;
  /** @type {string | undefined} */
  let refresh_token;

  if (t && typeof t === "object" && !Array.isArray(t)) {
    const tk = /** @type {Record<string, unknown>} */ (t);
    access_token =
      /** @type {string | undefined} */ (tk.access_token ?? tk.accessToken);
    refresh_token =
      /** @type {string | undefined} */ (tk.refresh_token ?? tk.refreshToken);
  }

  access_token =
    access_token ??
    /** @type {string | undefined} */ (b.access_token ?? b.accessToken);
  refresh_token =
    refresh_token ??
    /** @type {string | undefined} */ (b.refresh_token ?? b.refreshToken);

  if (typeof access_token !== "string" || !access_token.trim()) return null;
  if (typeof refresh_token !== "string" || !refresh_token.trim()) return null;

  return {
    access_token: access_token.trim(),
    refresh_token: refresh_token.trim(),
  };
}

/**
 * Locate user object inside varied shapes.
 * @param {unknown} bundle
 */
function extractUserRaw(bundle) {
  if (!bundle || typeof bundle !== "object") return undefined;
  /** @type {Record<string, unknown>} */
  const b = /** @type {Record<string, unknown>} */ (bundle);

  const u =
    b.user ??
    b.profile ??
    b.account ??
    (b.data && typeof b.data === "object" && !Array.isArray(b.data)
      ? /** @type {Record<string, unknown>} */ (b.data).user
      : undefined);
  return u;
}

/**
 * Returns a minimal safe user row for AuthContext / storage.
 * Accepts `/me` body, `{ user }` fragment, or full auth envelopes.
 * @param {unknown} raw
 */
export function normalizeUser(raw) {
  if (raw === null || raw === undefined) return null;

  const bundle = unwrapAuthBundle(raw);
  if (!bundle || typeof bundle !== "object" || Array.isArray(bundle)) return null;

  const extracted = extractUserRaw(bundle);
  let src =
    extracted !== undefined && extracted !== null ? extracted : null;

  if (!src || typeof src !== "object" || Array.isArray(src)) {
    /** Typical GET /auth/me payload (no nesting) — but not raw token payloads */
    if (
      "access_token" in bundle ||
      "accessToken" in bundle ||
      ("tokens" in bundle &&
        bundle.tokens !== undefined &&
        typeof bundle.tokens === "object")
    ) {
      return null;
    }
    /** @type {Record<string, unknown>} */
    const flat = bundle;
    if (
      flat.id != null ||
      flat.email != null ||
      flat.username != null ||
      flat.name != null
    ) {
      src = flat;
    } else {
      return null;
    }
  }

  /** @type {Record<string, unknown>} */
  const o = /** @type {Record<string, unknown>} */ (src);

  const nestedUser = o.user;
  if (
    nestedUser &&
    typeof nestedUser === "object" &&
    !Array.isArray(nestedUser)
  ) {
    return normalizeUser(nestedUser);
  }

  const id = o.id != null ? String(o.id) : "";
  const email = o.email != null ? String(o.email) : "";
  const username =
    o.username != null
      ? String(o.username)
      : o.name != null
        ? String(o.name)
        : "";
  const first_name =
    o.first_name != null
      ? String(o.first_name)
      : o.firstName != null
        ? String(o.firstName)
        : "";
  const last_name =
    o.last_name != null
      ? String(o.last_name)
      : o.lastName != null
        ? String(o.lastName)
        : "";
  const display_name =
    o.display_name != null
      ? String(o.display_name)
      : o.displayName != null
        ? String(o.displayName)
        : "";
  const avatar_url =
    o.avatar_url != null
      ? String(o.avatar_url)
      : o.avatarUrl != null
        ? String(o.avatarUrl)
        : "";

  const roleRaw = o.role ?? o.Role;
  let role = ROLES.USER;
  if (typeof roleRaw === "string") {
    const normalizedRole = roleRaw.trim().toLowerCase();
    const mappedRole =
      normalizedRole === "car_service" || normalizedRole === "service"
        ? ROLES.SERVICE
        : normalizedRole === "admin"
          ? ROLES.ADMIN
          : ROLES.USER;
    if (ALLOWED.has(mappedRole)) {
      role = mappedRole;
    }
  }

  const hasIdentity = Boolean(id || email || username);
  if (!hasIdentity) return null;

  return {
    id,
    email,
    username,
    first_name,
    last_name,
    display_name,
    avatar_url,
    role,
  };
}

/**
 * Full auth exchange / refresh response → storage shape.
 * @param {unknown} raw
 * @returns {{ tokens: { access_token: string, refresh_token: string }, user: ReturnType<typeof normalizeUser> } | null}
 */
export function normalizeAuthResponse(raw) {
  const bundle = unwrapAuthBundle(raw);
  if (!bundle || typeof bundle !== "object") return null;

  const tokens = extractTokens(bundle);
  if (!tokens) return null;

  const userRaw = extractUserRaw(bundle) ?? bundle;
  const user = normalizeUser(userRaw);
  if (!user) return null;

  return { tokens, user };
}
