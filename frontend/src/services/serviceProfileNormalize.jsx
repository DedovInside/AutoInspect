/**
 * Profile shapes: tolerate backend variance, keep UI contract (camelCase mock shape).
 */

import { serviceProfileDevLog } from "./serviceProfileDebug";

/** @param {unknown} raw */
export function coerceString(raw, fallback = "") {
  if (raw === null || raw === undefined) return fallback;
  const s = String(raw).trim();
  return s || fallback;
}

/**
 * @param {unknown} raw
 */
export function unwrapServiceProfileEnvelope(raw) {
  if (raw == null || typeof raw !== "object" || Array.isArray(raw)) return raw;

  /** @type {Record<string, unknown>} */
  let o = /** @type {Record<string, unknown>} */ (raw);

  if ("data" in o && typeof o.data === "object" && !Array.isArray(o.data)) {
    /** @type {Record<string, unknown>} */
    o = /** @type {Record<string, unknown>} */ (o.data);
  }

  if ("profile" in o && typeof o.profile === "object" && !Array.isArray(o.profile)) {
    return o.profile;
  }
  if (
    "service_profile" in o &&
    typeof o.service_profile === "object" &&
    !Array.isArray(o.service_profile)
  ) {
    return o.service_profile;
  }

  return o;
}

/** @param {unknown} raw */
export function normalizeLevelArray(raw) {
  if (raw == null) return [];

  /** @returns {unknown[]} */
  const asArray = Array.isArray(raw)
    ? raw
    : typeof raw === "string"
      ? raw.split(/[,;/|]/).map((s) => s.trim()).filter(Boolean)
      : [];

  const seen = new Set();
  const out = [];
  for (const item of asArray) {
    const lvl = coerceString(item, "");
    if (!lvl || seen.has(lvl)) continue;
    seen.add(lvl);
    out.push(lvl);
  }

  return out;
}

/**
 * One offering row → { type, levels } expected by UI.
 * @param {unknown} raw
 */
export function normalizeServiceEntry(raw) {
  if (raw == null || typeof raw !== "object" || Array.isArray(raw)) {
    return { type: "", levels: [] };
  }

  /** @type {Record<string, unknown>} */
  const o = /** @type {Record<string, unknown>} */ (raw);

  const type = coerceString(
    o.type ?? o.service_type ?? o.kind ?? o.name ?? o.slug,
    ""
  );

  const levels = normalizeLevelArray(
    o.levels ??
      o.severity_levels ??
      o.severityLevels ??
      o.severities ??
      o.damage_levels ??
      o.damageLevels
  );

  if (!type && import.meta.env.DEV) {
    serviceProfileDevLog("normalize.entry_drop_empty_type", {});
  }

  return { type, levels };
}

/**
 * @param {unknown} rawArray
 */
export function normalizeServicesList(rawArray) {
  let list;

  if (Array.isArray(rawArray)) list = rawArray;
  else if (rawArray == null || rawArray === undefined) list = [];
  else if (typeof rawArray === "object") {
    /** @type {Record<string, unknown>} */
    const o = /** @type {Record<string, unknown>} */ (rawArray);
    /** @type {unknown} */
    const inner =
      o.services ?? o.offerings ?? o.capabilities ?? o.items ?? [];
    list = Array.isArray(inner) ? inner : [];
  } else {
    list = [];
  }

  if (!Array.isArray(list)) return [];

  return list
    .map((entry) => normalizeServiceEntry(entry))
    .filter((r) => r.type);
}

/**
 * @param {unknown} raw
 */
export function normalizeServiceProfile(raw) {
  const env = unwrapServiceProfileEnvelope(raw);

  if (env == null || typeof env !== "object" || Array.isArray(env)) {
    if (env != null && import.meta.env.DEV) {
      serviceProfileDevLog("normalize.blank_envelope", { kind: typeof env });
    }
    return {
      name: "",
      phone: "",
      address: "",
      description: "",
      services: [],
    };
  }

  /** @type {Record<string, unknown>} */
  const o = /** @type {Record<string, unknown>} */ (env);

  /** @type {unknown} */
  const servicesSrc =
    o.services ??
    o.offerings ??
    o.capabilities ??
    (Array.isArray(o.service_list)
      ? o.service_list
      : null);

  let servicesArrayCandidate = [];

  if (Array.isArray(servicesSrc)) {
    servicesArrayCandidate = servicesSrc;
  } else if (
    servicesSrc != null &&
    typeof servicesSrc === "object"
  ) {
    /** @type {Record<string, unknown>} */
    const bag = /** @type {Record<string, unknown>} */ (servicesSrc);
    /** @type {unknown} */
    const inner =
      bag.items ??
      bag.results ??
      bag.data ??
      (Array.isArray(bag.entries) ? bag.entries : null);
    servicesArrayCandidate = Array.isArray(inner) ? inner : [];
  }

  const services = normalizeServicesList(servicesArrayCandidate);

  if (
    ("services" in o || "offerings" in o || "capabilities" in o) &&
    services.length === 0 &&
    import.meta.env.DEV
  ) {
    serviceProfileDevLog("normalize.services_fallback_empty", {});
  }

  return {
    name: coerceString(o.name ?? o.title ?? o.organization_name ?? o.organizationName, ""),
    phone: coerceString(
      o.phone ?? o.contact_phone ?? o.phone_number ?? o.contactPhone ?? o.contact_phone_number,
      ""
    ),
    address: coerceString(
      o.address ?? o.location ?? o.full_address ?? o.fullAddress,
      ""
    ),
    description: coerceString(
      o.description ?? o.about ?? o.notes ?? o.summary,
      ""
    ),
    services,
  };
}

/**
 * Outbound PUT / local persist shape (minimal, cleaned).
 * Mirrors current mock fields; callers may map to snake_case API if needed downstream.
 *
 * @param {unknown} profile
 */
export function normalizeServiceProfilePayload(profile) {
  const p =
    typeof profile === "object" &&
    profile !== null &&
    !Array.isArray(profile)
      ? /** @type {Record<string, unknown>} */ (profile)
      : {};

  const servicesSrc = normalizeServicesList(
    p.services ?? p.offerings ?? []
  ).map((s) => ({
    type: coerceString(s.type, ""),
    levels: normalizeLevelArray(s.levels ?? s.severities ?? s.severity_levels),
  }));

  return {
    name: coerceString(p.name ?? p.title, "").slice(0, 500),
    phone: coerceString(
      p.phone ?? p.contact_phone ?? p.phone_number,
      ""
    ).slice(0, 80),
    address: coerceString(
      p.address ?? p.location,
      ""
    ).slice(0, 500),
    description: coerceString(
      p.description ?? p.about ?? p.notes ?? p.summary,
      ""
    ).slice(0, 2000),
    services: servicesSrc.filter((s) => s.type),
  };
}

/**
 * Merge server partial response onto what we submitted (canonical for form state).
 *
 * @param {unknown} clientSent
 * @param {unknown} serverRaw
 */
export function mergeServiceProfileResponse(clientSent, serverRaw) {
  const base = normalizeServiceProfile(clientSent ?? {});
  const patch = normalizeServiceProfile(serverRaw ?? {});

  const pickLevels = patch.services.length > 0 ? patch.services : base.services;

  return normalizeServiceProfile({
    name: coerceString(patch.name, "") || base.name,
    phone: coerceString(patch.phone, "") || base.phone,
    address: coerceString(patch.address, "") || base.address,
    description:
      patch.description.trim() !== ""
        ? patch.description
        : base.description,
    services: pickLevels,
  });
}
