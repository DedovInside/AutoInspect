/**
 * Service registration + ML training request shapes (home submit + admin list).
 * Primary contract matches existing mock rows + HomePage form fields.
 */

import { homeFormsDevLog } from "./homeFormsDebug";

/** @param {unknown} raw */
export function coerceString(raw, fallback = "") {
  if (raw === null || raw === undefined) return fallback;
  const s = String(raw).trim();
  return s || fallback;
}

/** @param {unknown} raw */
export function unwrapRecordEnvelope(raw) {
  if (raw == null || typeof raw !== "object" || Array.isArray(raw)) return raw;

  /** @type {Record<string, unknown>} */
  let o = /** @type {Record<string, unknown>} */ (raw);

  if ("data" in o && typeof o.data === "object" && !Array.isArray(o.data)) {
    o = /** @type {Record<string, unknown>} */ (o.data);
  }

  if ("registration" in o && typeof o.registration === "object") {
    return o.registration;
  }
  if ("service_registration" in o && typeof o.service_registration === "object") {
    return o.service_registration;
  }
  if ("training_request" in o && typeof o.training_request === "object") {
    return o.training_request;
  }
  if ("request" in o && typeof o.request === "object") {
    return o.request;
  }

  return o;
}

/**
 * Pull list arrays from envelopes without logging payloads.
 *
 * @param {unknown} raw
 * @param {string[]} keys
 */
export function extractItemsArray(raw, keys) {
  if (Array.isArray(raw)) return raw;
  if (!raw || typeof raw !== "object") return [];

  /** @type {Record<string, unknown>} */
  const o = /** @type {Record<string, unknown>} */ (raw);

  for (const k of keys) {
    /** @type {unknown} */
    const v = o[k];
    if (Array.isArray(v)) return v;
  }

  /** @type {unknown} */
  const data = o.data;
  if (data && typeof data === "object" && !Array.isArray(data)) {
    /** @type {Record<string, unknown>} */
    const d = /** @type {Record<string, unknown>} */ (data);
    for (const k of keys) {
      /** @type {unknown} */
      const v = d[k];
      if (Array.isArray(v)) return v;
    }
  }

  return [];
}

/**
 * @param {unknown} raw
 */
export function normalizeServiceRegistrationStatus(raw) {
  const s = coerceString(raw, "").toLowerCase();
  if (["pending", "new", "submitted", "open"].includes(s)) return "pending";
  if (["approved", "accepted"].includes(s)) return "approved";
  if (["rejected", "declined", "denied"].includes(s)) return "rejected";
  if (s) return s;
  return "pending";
}

/**
 * @param {unknown} raw
 */
export function normalizeServiceRegistration(raw) {
  const e = unwrapRecordEnvelope(raw);

  if (e == null || typeof e !== "object" || Array.isArray(e)) {
    if (e != null && import.meta.env.DEV) {
      homeFormsDevLog("registration", "normalize.blank", { kind: typeof e });
    }
    return {
      id: "",
      organization: "",
      city: "",
      address: "",
      contact_name: "",
      contact_phone: "",
      contact_email: "",
      description: "",
      submitted_at: "",
      status: "pending",
      rejection_reason: "",
    };
  }

  /** @type {Record<string, unknown>} */
  const o = /** @type {Record<string, unknown>} */ (e);

  const id = coerceString(o.id ?? o.registration_id ?? o.registrationId, "");
  const organization = coerceString(
    o.organization ?? o.company_name ?? o.companyName ?? o.name ?? o.title,
    ""
  );
  const city = coerceString(o.city ?? o.town, "");
  const address = coerceString(
    o.address ?? o.location ?? o.street_address ?? o.streetAddress ?? o.full_address,
    ""
  );
  const contact_name = coerceString(
    o.contact_name ?? o.contactName ?? o.contact_person,
    ""
  );
  const contact_phone = coerceString(
    o.contact_phone ?? o.phone ?? o.contactPhone ?? o.phone_number,
    ""
  );
  const contact_email = coerceString(
    o.contact_email ?? o.email ?? o.contactEmail,
    ""
  );
  const description = coerceString(
    o.description ?? o.notes ?? o.comment ?? o.summary,
    ""
  );
  const submitted_at = coerceString(
    o.submitted_at ?? o.submittedAt ?? o.created_at ?? o.createdAt,
    ""
  );
  const status = normalizeServiceRegistrationStatus(o.status ?? o.state);
  const rejection_reason = coerceString(
    o.rejection_reason ?? o.rejectionReason ?? o.reject_reason,
    ""
  );

  return {
    id,
    organization,
    city,
    address,
    contact_name,
    contact_phone,
    contact_email,
    description,
    submitted_at,
    status,
    rejection_reason,
  };
}

/**
 * Home form → cleaned payload (UI keys). `city` reserved for future / API split.
 *
 * @param {unknown} form
 */
export function normalizeServiceRegistrationPayload(form) {
  const f =
    typeof form === "object" && form !== null && !Array.isArray(form)
      ? /** @type {Record<string, unknown>} */ (form)
      : {};

  const organization = coerceString(
    f.organization ?? f.company_name ?? f.companyName,
    ""
  ).slice(0, 300);
  const address = coerceString(
    f.address ?? f.location,
    ""
  ).slice(0, 500);
  const phoneRaw = coerceString(
    f.phone ?? f.contact_phone ?? f.contactPhone,
    ""
  );
  const phone = phoneRaw
    .replace(/[^\d+()\-.\s]/g, "")
    .trim()
    .slice(0, 40);
  const description = coerceString(
    f.description ?? f.notes,
    ""
  ).slice(0, 2000);
  const city = coerceString(f.city, "").slice(0, 120);

  if (!organization && !address && import.meta.env.DEV) {
    homeFormsDevLog("registration", "payload.empty_core", {});
  }

  return {
    organization,
    city,
    address,
    phone,
    description,
  };
}

/**
 * Serialize for POST `/v1/service-registrations`.
 *
 * @param {unknown} form
 */
export function serviceRegistrationPayloadToApiBody(form) {
  const p = normalizeServiceRegistrationPayload(form);
  return {
    organization: p.organization,
    city: p.city || undefined,
    address: p.address,
    contact_phone: p.phone,
    description: p.description || undefined,
  };
}

/** @param {unknown} sentForm */
/** @param {unknown} serverRaw */
export function mergeServiceRegistrationResponse(sentForm, serverRaw) {
  const sent = normalizeServiceRegistrationPayload(sentForm);
  const srv = normalizeServiceRegistration(serverRaw ?? {});

  let merged = normalizeServiceRegistration({
    organization: srv.organization || sent.organization,
    city: srv.city || sent.city,
    address: srv.address || sent.address,
    contact_phone: srv.contact_phone || sent.phone,
    description:
      coerceString(srv.description, "").trim() !== ""
        ? srv.description
        : sent.description,
    id: coerceString(srv.id, ""),
    submitted_at: srv.submitted_at,
    status: srv.status,
    contact_email: srv.contact_email,
    contact_name: srv.contact_name,
    rejection_reason: srv.rejection_reason,
  });

  if (!coerceString(merged.id, "")) {
    if (import.meta.env.DEV) {
      homeFormsDevLog("registration", "merge.fallback_id", {});
    }
    merged = normalizeServiceRegistration({
      ...merged,
      id: `sr-pending-${Date.now()}`,
    });
  }

  return merged;
}

/**
 * @param {unknown} raw
 */
export function normalizeTrainingRequestStatus(raw) {
  const s = coerceString(raw, "").toLowerCase();
  if (["pending", "new", "submitted"].includes(s)) return "pending";
  if (["approved", "accepted"].includes(s)) return "approved";
  if (["rejected", "declined"].includes(s)) return "rejected";
  if (["completed", "done"].includes(s)) return "completed";
  if (s) return s;
  return "pending";
}

/** @param {unknown} raw */
export function coerceTrainingYear(raw) {
  const s = coerceString(raw, "");
  const digits = s.replace(/\D/g, "");
  if (digits.length >= 4) return digits.slice(0, 4);
  if (s) return s.slice(0, 32);
  return "";
}

/**
 * @param {unknown} raw
 */
export function normalizeTrainingRequest(raw) {
  const e = unwrapRecordEnvelope(raw);

  if (e == null || typeof e !== "object" || Array.isArray(e)) {
    if (e != null && import.meta.env.DEV) {
      homeFormsDevLog("training", "normalize.blank", { kind: typeof e });
    }
    return {
      id: "",
      brand: "",
      model: "",
      generation: "",
      year: "",
      years: "",
      description: "",
      submitted_at: "",
      status: "pending",
      submitted_by: "",
    };
  }

  /** @type {Record<string, unknown>} */
  const o = /** @type {Record<string, unknown>} */ (e);

  const id = coerceString(o.id ?? o.request_id ?? o.requestId, "");
  const brand = coerceString(o.brand ?? o.make, "");
  const model = coerceString(o.model, "");
  const generation = coerceString(
    o.generation ?? o.gen ?? o.series,
    ""
  );

  const year = coerceTrainingYear(o.year ?? o.years ?? o.model_year);
  const submitted_at = coerceString(
    o.submitted_at ?? o.submittedAt ?? o.created_at ?? o.createdAt,
    ""
  );
  const status = normalizeTrainingRequestStatus(o.status ?? o.state);
  const description = coerceString(
    o.description ?? o.notes ?? o.reason,
    ""
  );
  const submitted_by = coerceString(
    o.submitted_by ?? o.submittedBy ?? o.user_id ?? o.requester_id,
    ""
  );

  return {
    id,
    brand,
    model,
    generation,
    year,
    years: year,
    description,
    submitted_at,
    status,
    submitted_by,
  };
}

/**
 * @param {unknown} form
 */
export function normalizeTrainingRequestPayload(form) {
  const f =
    typeof form === "object" && form !== null && !Array.isArray(form)
      ? /** @type {Record<string, unknown>} */ (form)
      : {};

  const brand = coerceString(f.brand ?? f.make, "").slice(0, 120);
  const model = coerceString(f.model, "").slice(0, 120);
  const generation = coerceString(f.generation ?? f.gen, "").slice(
    0,
    80
  );
  const year = coerceTrainingYear(f.year ?? f.years ?? f.model_year);
  const description = coerceString(f.description ?? f.notes, "").slice(
    0,
    2000
  );

  return {
    brand,
    model,
    generation,
    year,
    description,
  };
}

/** POST `/v1/training-requests` */
export function trainingRequestPayloadToApiBody(form) {
  const p = normalizeTrainingRequestPayload(form);
  return {
    brand: p.brand,
    model: p.model,
    generation: p.generation,
    year: p.year,
    description: p.description || undefined,
  };
}

/** @param {unknown} sentForm */
/** @param {unknown} serverRaw */
export function mergeTrainingRequestResponse(sentForm, serverRaw) {
  const sent = normalizeTrainingRequestPayload(sentForm);
  const srv = normalizeTrainingRequest(serverRaw ?? {});
  const y =
    coerceString(srv.year, "").trim() !== ""
      ? srv.year
      : coerceString(srv.years, "").trim() !== ""
        ? srv.years
        : sent.year;

  let merged = normalizeTrainingRequest({
    ...srv,
    brand: srv.brand || sent.brand,
    model: srv.model || sent.model,
    generation: srv.generation || sent.generation,
    year: y,
    years: y,
    description:
      coerceString(srv.description, "").trim() !== ""
        ? srv.description
        : sent.description,
  });

  if (!coerceString(merged.id, "")) {
    if (import.meta.env.DEV) {
      homeFormsDevLog("training", "merge.fallback_id", {});
    }
    merged = normalizeTrainingRequest({
      ...merged,
      id: `tr-pending-${Date.now()}`,
    });
  }

  return merged;
}

/**
 * @param {unknown} raw
 */
export function normalizeServiceRegistrationList(raw) {
  const list = extractItemsArray(raw, [
    "items",
    "results",
    "registrations",
    "requests",
    "data",
  ]);

  return list
    .map((row) => normalizeServiceRegistration(row))
    .filter((r) => r.id);
}

/**
 * @param {unknown} raw
 */
export function normalizeTrainingRequestList(raw) {
  const list = extractItemsArray(raw, [
    "items",
    "results",
    "requests",
    "training_requests",
    "data",
  ]);

  return list
    .map((row) => normalizeTrainingRequest(row))
    .filter((r) => r.id);
}
