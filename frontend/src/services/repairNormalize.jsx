/**
 * Repair request shapes: backend variance + safe fallbacks for UI.
 * Primary contract matches existing mock rows (snake_case).
 */

import { repairDevLog } from "./repairDebug";

/** @param {unknown} raw */
function coerceString(raw, fallback = "") {
  if (raw === null || raw === undefined) return fallback;
  const s = String(raw).trim();
  return s || fallback;
}

/**
 * @param {unknown} raw
 */
export function unwrapRepairEnvelope(raw) {
  if (raw === null || raw === undefined) return raw;
  if (typeof raw !== "object" || Array.isArray(raw)) return raw;

  /** @type {Record<string, unknown>} */
  const top = /** @type {Record<string, unknown>} */ (raw);

  if (
    top.data !== undefined &&
    typeof top.data === "object" &&
    !Array.isArray(top.data)
  ) {
    const d = /** @type {Record<string, unknown>} */ (top.data);
    if (d.request !== undefined && typeof d.request === "object") {
      return d.request;
    }
    return top.data;
  }

  if (top.request !== undefined && typeof top.request === "object") {
    return top.request;
  }

  return top;
}

/**
 * @param {unknown} raw
 */
export function normalizeRepairRequestStatus(raw) {
  const s = coerceString(raw, "")
    .toLowerCase()
    .replace(/\s+/g, "_");

  switch (s) {
    case "pending":
    case "new":
    case "submitted":
    case "open":
      return "pending";
    case "accepted":
    case "approved":
    case "confirmed":
      return "accepted";
    case "rejected":
    case "declined":
    case "cancelled":
    case "canceled":
    case "denied":
      return "rejected";
    default:
      repairDevLog("normalize.status_fallback", { rawValue: raw });
      return "pending";
  }
}

/** @param {unknown} raw */
export function normalizeRepairService(raw) {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
    return { name: "—", phone: "—", address: "—" };
  }

  /** @type {Record<string, unknown>} */
  const o = /** @type {Record<string, unknown>} */ (raw);

  return {
    name: coerceString(o.name ?? o.title ?? o.organization, "—"),
    phone: coerceString(
      o.phone ?? o.phone_number ?? o.contact_phone ?? o.contactPhone,
      "—"
    ),
    address: coerceString(o.address ?? o.location ?? o.full_address ?? o.fullAddress, "—"),
  };
}

/** @param {unknown} raw */
export function normalizeRepairUserContact(raw) {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return null;

  /** @type {Record<string, unknown>} */
  const o = /** @type {Record<string, unknown>} */ (raw);

  const email = coerceString(
    o.email ?? o.contact_email ?? o.contactEmail,
    ""
  );
  if (!email) return null;
  return { email };
}

/**
 * Single repair request row (USER or SERVICE list / detail).
 * @param {unknown} raw
 * @param {{
 *   defaults?: Partial<{ analysis_id?: string; car_brand?: string; damage_summary?: string; service_id?: string }>
 * }} [ctx]
 */
export function normalizeRepairRequest(raw, ctx = {}) {
  const defaults = ctx.defaults ?? {};

  const e = unwrapRepairEnvelope(raw);
  if (!e || typeof e !== "object" || Array.isArray(e)) {
    repairDevLog("normalize.blank_envelope", {});
    return {
      id: "",
      created_at: new Date().toISOString(),
      car_brand: coerceString(defaults.car_brand, "—"),
      analysis_id: coerceString(defaults.analysis_id, ""),
      service: normalizeRepairService(null),
      status: "pending",
      note: "",
      damage_summary: coerceString(defaults.damage_summary, "—"),
      severity: "—",
      image_url: "",
      user: null,
      rejection_reason: "",
    };
  }

  /** @type {Record<string, unknown>} */
  const o = /** @type {Record<string, unknown>} */ (e);

  const id = coerceString(o.id ?? o.request_id ?? o.requestId, "");
  const created_at = coerceString(
    o.created_at ?? o.createdAt,
    new Date().toISOString()
  );
  const car_brand = coerceString(
    o.car_brand ?? o.carBrand ?? o.vehicle_brand ?? o.vehicleBrand ?? defaults.car_brand,
    "—"
  );
  const analysis_id = coerceString(
    o.analysis_id ?? o.analysisId ?? defaults.analysis_id,
    ""
  );

  const serviceSource =
    o.service ??
    o.service_info ??
    o.serviceInfo ??
    o.autoservice ??
    o.workshop;
  const service = normalizeRepairService(serviceSource);

  const status = normalizeRepairRequestStatus(
    o.status ?? o.state ?? o.lifecycle
  );

  const note = coerceString(o.note ?? o.message ?? o.comment, "");
  const damage_summary = coerceString(
    o.damage_summary ?? o.damageSummary ?? o.summary ?? defaults.damage_summary,
    "—"
  );

  const severity = coerceString(
    o.severity ?? o.overall_severity ?? o.overallSeverity,
    "—"
  );

  const image_url = coerceString(o.image_url ?? o.imageUrl, "");

  const user =
    normalizeRepairUserContact(o.user ?? o.client ?? o.customer) ?? null;

  const rejection_reason = coerceString(
    o.rejection_reason ?? o.rejectionReason ?? o.reject_reason,
    ""
  );

  return {
    id,
    created_at,
    car_brand,
    analysis_id,
    service,
    status,
    note,
    damage_summary,
    severity,
    image_url,
    user,
    rejection_reason,
  };
}

/** @param {unknown} raw */
export function normalizeRepairRequestList(raw) {
  let list = null;

  if (Array.isArray(raw)) {
    list = raw;
  } else if (raw && typeof raw === "object") {
    /** @type {Record<string, unknown>} */
    const o = /** @type {Record<string, unknown>} */ (raw);
    const inner =
      o.items ??
      o.results ??
      o.requests ??
      o.data ??
      (o.data &&
      typeof o.data === "object" &&
      !Array.isArray(o.data) &&
      /** @type {Record<string, unknown>} */ (o.data).items
        ? /** @type {Record<string, unknown>} */ (o.data).items
        : null);

    if (Array.isArray(inner)) list = inner;
  }

  if (!Array.isArray(list)) return [];

  return list
    .map((item) => normalizeRepairRequest(item))
    .filter((row) => row.id);
}

/**
 * Validates create payload; returns snake_case keys for POST.
 * Extra fields allowed on payload for mocks / enrichment but not sent unless API extended.
 */
export function normalizeCreateRepairPayload(payload) {
  const p = payload ?? {};
  const analysis_id = coerceString(p.analysis_id ?? p.analysisId, "");
  const service_id = coerceString(p.service_id ?? p.serviceId, "");
  if (!analysis_id || !service_id) {
    throw new Error("Не указан анализ или автосервис для заявки");
  }

  const car_brand = coerceString(p.car_brand ?? p.carBrand, "");
  const damage_summary = coerceString(
    p.damage_summary ?? p.damageSummary,
    ""
  );

  return {
    analysis_id,
    service_id,
    /** Optional enrichment (UI / mocks), not sent in minimal API POST */
    car_brand: car_brand || undefined,
    damage_summary: damage_summary || undefined,
  };
}

/** Merge server PATCH into previous row without losing unchanged nested fields */
export function mergeRepairRequest(prev, serverNorm) {
  if (!prev) return serverNorm ?? prev;
  if (!serverNorm || typeof serverNorm !== "object" || Array.isArray(serverNorm)) {
    return prev;
  }

  const id = coerceString(serverNorm.id ?? prev.id, "");
  if (!id) return prev;

  const svcPrev = prev.service ?? normalizeRepairService(null);
  const svcNext =
    typeof serverNorm.service === "object" &&
    serverNorm.service !== null &&
    !Array.isArray(serverNorm.service)
      ? normalizeRepairService(serverNorm.service)
      : null;

  return {
    ...prev,
    ...serverNorm,
    id,
    service: svcNext ? { ...svcPrev, ...svcNext } : svcPrev,
    user: normalizeRepairUserContact(serverNorm.user) ?? prev.user ?? null,
  };
}
