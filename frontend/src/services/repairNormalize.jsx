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
    case "denied":
      return "rejected";
    case "cancelled":
    case "canceled":
      return "canceled";
    default:
      repairDevLog("normalize.status_fallback", { rawValue: raw });
      return "pending";
  }
}

/** @param {unknown} raw */
export function normalizeRepairService(raw) {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
    return { id: "", name: "", city: "", phone: "", email: "", address: "" };
  }

  /** @type {Record<string, unknown>} */
  const o = /** @type {Record<string, unknown>} */ (raw);

  return {
    id: coerceString(o.id ?? o.profile_id ?? o.profileId, ""),
    name: coerceString(
      o.organization_name ??
        o.organizationName ??
        o.name ??
        o.title ??
        o.organization,
      ""
    ),
    city: coerceString(o.city ?? o.town, ""),
    phone: coerceString(
      o.phone ?? o.phone_number ?? o.contact_phone ?? o.contactPhone,
      ""
    ),
    email: coerceString(o.email ?? o.contact_email ?? o.contactEmail, ""),
    address: coerceString(o.address ?? o.location ?? o.full_address ?? o.fullAddress, ""),
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
  const phone = coerceString(
    o.phone ?? o.contact_phone ?? o.contactPhone ?? o.customer_phone ?? o.customerPhone,
    ""
  );
  const name = coerceString(
    o.name ?? o.full_name ?? o.fullName ?? o.customer_name ?? o.customerName,
    ""
  );
  if (!email && !phone && !name) return null;
  return { email, phone, name };
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
      severity: "",
      image_url: "",
      user: null,
      rejection_reason: "",
      analysis: null,
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
    o.car_brand ??
      o.carBrand ??
      o.vehicle_brand ??
      o.vehicleBrand ??
      o.car_make ??
      (o.analysis && typeof o.analysis === "object"
        ? /** @type {Record<string, unknown>} */ (o.analysis).car_make
        : undefined) ??
      defaults.car_brand,
    ""
  );
  const analysis_id = coerceString(
    o.analysis_id ?? o.analysisId ?? o.analysis_job_id ?? o.analysisJobID ?? defaults.analysis_id,
    ""
  );

  const serviceSource =
    o.car_service_profile ??
    o.carServiceProfile ??
    o.service ??
    o.service_info ??
    o.serviceInfo ??
    o.autoservice ??
    o.workshop;
  const service = normalizeRepairService(serviceSource);

  const status = normalizeRepairRequestStatus(
    o.status ?? o.state ?? o.lifecycle
  );

  const note = coerceString(o.note ?? o.message ?? o.comment ?? o.service_comment, "");
  const customer_comment = coerceString(o.customer_comment ?? o.customerComment, "");
  const service_comment = coerceString(o.service_comment ?? o.serviceComment, "");
  const damage_summary = coerceString(
    o.damage_summary ??
      o.damageSummary ??
      o.summary ??
      repairSummaryToText(o.repair_summary) ??
      defaults.damage_summary,
    ""
  );

  const severity = coerceString(
    o.severity ?? o.overall_severity ?? o.overallSeverity,
    ""
  );

  const image_url = coerceString(o.image_url ?? o.imageUrl, "");

  const user =
    normalizeRepairUserContact(o.user ?? o.client ?? o.customer) ??
    normalizeRepairUserContact({
      name: o.customer_name,
      phone: o.customer_phone,
      email: o.customer_email,
    }) ??
    null;

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
    customer_comment,
    service_comment,
    damage_summary,
    severity,
    image_url,
    user,
    rejection_reason,
    analysis: normalizeRepairAnalysis(o.analysis),
    repair_summary: Array.isArray(o.repair_summary) ? o.repair_summary : [],
    service_estimate: Array.isArray(o.service_estimate) ? o.service_estimate : [],
    estimated_price_min: o.estimated_price_min,
    estimated_price_max: o.estimated_price_max,
  };
}

/** @param {unknown} raw */
function normalizeRepairAnalysis(raw) {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return null;

  /** @type {Record<string, unknown>} */
  const o = /** @type {Record<string, unknown>} */ (raw);
  return {
    id: coerceString(o.id ?? o.analysis_id ?? o.analysisJobID, ""),
    car_make: coerceString(o.car_make ?? o.carMake ?? o.make, ""),
    car_model: coerceString(o.car_model ?? o.carModel ?? o.model, ""),
    car_generation: coerceString(o.car_generation ?? o.carGeneration ?? o.generation, ""),
    car_year: coerceString(o.car_year ?? o.carYear ?? o.year, ""),
    completed_at: coerceString(o.completed_at ?? o.completedAt, ""),
    requested_at: coerceString(o.requested_at ?? o.requestedAt ?? o.created_at, ""),
  };
}

function repairSummaryToText(summary) {
  if (!Array.isArray(summary) || summary.length === 0) return "";
  return summary
    .map((item) => {
      if (!item || typeof item !== "object" || Array.isArray(item)) return "";
      const o = /** @type {Record<string, unknown>} */ (item);
      const part = coerceString(o.part_name_ru ?? o.part_name, "деталь");
      const side = coerceString(o.side_ru ?? o.side, "");
      const damages = Array.isArray(o.damage_types)
        ? o.damage_types
            .map((d) => {
              if (!d || typeof d !== "object" || Array.isArray(d)) return "";
              const dd = /** @type {Record<string, unknown>} */ (d);
              const name = coerceString(dd.name_ru ?? dd.code, "повреждение");
              const count = Number(dd.count ?? 0);
              return Number.isFinite(count) && count > 1 ? `${name} ×${count}` : name;
            })
            .filter(Boolean)
            .join(", ")
        : "";
      return `${part}${side ? `, ${side}` : ""}${damages ? `: ${damages}` : ""}`;
    })
    .filter(Boolean)
    .join("; ");
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
  const customer_name = coerceString(p.customer_name ?? p.customerName, "");
  const customer_phone = coerceString(p.customer_phone ?? p.customerPhone, "");
  const customer_email = coerceString(p.customer_email ?? p.customerEmail, "");
  const customer_comment = coerceString(p.customer_comment ?? p.customerComment, "");

  return {
    analysis_id,
    service_id,
    customer_name: customer_name || undefined,
    customer_phone: customer_phone || undefined,
    customer_email: customer_email || undefined,
    customer_comment: customer_comment || undefined,
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
