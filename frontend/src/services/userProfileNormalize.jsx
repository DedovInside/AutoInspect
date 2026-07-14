/**
 * User profile shapes for GET/PATCH /v1/auth/me.
 */

/** @param {unknown} raw */
export function coerceString(raw, fallback = "") {
    if (raw === null || raw === undefined) return fallback;
    const s = String(raw).trim();
    return s || fallback;
  }
  
  /** @param {unknown} raw */
  function optionalString(raw) {
    if (raw === null || raw === undefined) return "";
    return String(raw).trim();
  }
  
  /**
   * @param {unknown} raw
   */
  export function normalizeUserProfile(raw) {
    if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
      return {
        id: "",
        email: "",
        username: "",
        first_name: "",
        last_name: "",
        contact_name: "",
        contact_phone: "",
        contact_email: "",
        avatar_url: "",
        display_name: "",
        role: "USER",
      };
    }
  
    /** @type {Record<string, unknown>} */
    const o = /** @type {Record<string, unknown>} */ (raw);
  
    const roleRaw = coerceString(o.role ?? o.Role, "user").toLowerCase();
    let role = "USER";
    if (roleRaw === "car_service" || roleRaw === "service") role = "SERVICE";
    else if (roleRaw === "admin") role = "ADMIN";
  
    return {
      id: coerceString(o.id, ""),
      email: coerceString(o.email, ""),
      username: coerceString(o.username ?? o.name, ""),
      first_name: optionalString(o.first_name ?? o.firstName),
      last_name: optionalString(o.last_name ?? o.lastName),
      contact_name: optionalString(o.contact_name ?? o.contactName),
      contact_phone: optionalString(o.contact_phone ?? o.contactPhone ?? o.phone),
      contact_email: optionalString(o.contact_email ?? o.contactEmail),
      avatar_url: optionalString(o.avatar_url ?? o.avatarUrl),
      display_name: optionalString(o.display_name ?? o.displayName),
      role,
    };
  }
  
  /**
   * @param {{ contact_name?: string, contact_phone?: string, contact_email?: string }} profile
   */
  export function userProfileToApiBody(profile) {
    const body = {};
  
    if (profile.contact_name !== undefined) {
      const value = coerceString(profile.contact_name, "");
      body.contact_name = value || null;
    }
    if (profile.contact_phone !== undefined) {
      const value = coerceString(profile.contact_phone, "");
      body.contact_phone = value || null;
    }
    if (profile.contact_email !== undefined) {
      const value = coerceString(profile.contact_email, "");
      body.contact_email = value || null;
    }
  
    return body;
  }
  
  /**
   * @param {ReturnType<typeof normalizeUserProfile>} profile
   */
  export function resolveProfileDisplayName(profile) {
    if (profile?.contact_name) return profile.contact_name;
    if (profile?.display_name) return profile.display_name;
    const parts = [profile?.first_name, profile?.last_name].filter(Boolean);
    if (parts.length > 0) return parts.join(" ");
    if (profile?.username) return profile.username;
    if (profile?.email) return profile.email;
    return "Пользователь";
  }
  