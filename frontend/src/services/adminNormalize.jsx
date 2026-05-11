/**
 * Admin panel normalization: ML models, service registrations (queue), training requests.
 * Keeps legacy UI field names (`years`, `deprecated` for deactivated models).
 */

import { normalizeApiError } from "./apiFoundation";
import { adminDevLog } from "./adminDebug";
import {
  coerceString,
  extractItemsArray,
  normalizeServiceRegistration,
  normalizeTrainingRequest,
} from "./homeFormsNormalize";

/** @param {unknown} err */
export function rethrowAbortOrWrapAdminError(err) {
  const n = normalizeApiError(err);
  adminDevLog("mutation.fail", { status: n.status, code: n.code });
  if (n.code === "aborted") {
    throw err instanceof Error ? err : new Error(n.message || "Запрос отменён");
  }
  if (n.status === 422 || n.status === 400) {
    throw new Error(n.message || "Ошибка валидации");
  }
  if (n.status === 409) {
    throw new Error(n.message || "Данные устарели, обновите список");
  }
  if (n.status === 0) {
    throw new Error(n.message || "Не удалось связаться с сервером");
  }
  throw new Error(
    typeof n.message === "string" && n.message.trim()
      ? n.message
      : "Ошибка запроса"
  );
}

/* ---- ML models ---- */

/** @param {unknown} raw */
export function normalizeMLModelStatus(raw) {
  const s = coerceString(raw, "").toLowerCase().replace(/\s+/g, "_");
  if (["active", "enabled", "online"].includes(s)) return "active";
  if (
    ["inactive", "disabled", "offline", "deprecated", "retired", "archived"].includes(
      s
    )
  ) {
    return "deprecated";
  }
  if (s) return s;
  return "deprecated";
}

/** @param {unknown} raw */
function unwrapMLModel(raw) {
  if (raw == null || typeof raw !== "object" || Array.isArray(raw)) return raw;

  /** @type {Record<string, unknown>} */
  let o = /** @type {Record<string, unknown>} */ (raw);

  if ("data" in o && typeof o.data === "object" && !Array.isArray(o.data)) {
    o = /** @type {Record<string, unknown>} */ (o.data);
  }

  if ("model" in o && typeof o.model === "object" && !Array.isArray(o.model)) {
    return o.model;
  }
  if ("ml_model" in o && typeof o.ml_model === "object") {
    return o.ml_model;
  }

  return o;
}

/**
 * @param {unknown} raw
 */
export function normalizeMLModel(raw) {
  const e = unwrapMLModel(raw);

  if (e == null || typeof e !== "object" || Array.isArray(e)) {
    if (e != null && import.meta.env.DEV) {
      adminDevLog("ml.normalize.blank", { kind: typeof e });
    }
    return {
      id: "",
      brand: "",
      model: "",
      generation: "",
      years: "",
      version: "",
      file: "",
      parts_catalog: "",
      accuracy: null,
      created_at: "",
      status: "deprecated",
    };
  }

  /** @type {Record<string, unknown>} */
  const o = /** @type {Record<string, unknown>} */ (e);

  let accuracy =
    o.accuracy ??
    o.accuracy_score ??
    o.metric_accuracy ??
    o.score ??
    null;
  if (accuracy != null && typeof accuracy !== "number") {
    const n = Number(accuracy);
    accuracy = Number.isFinite(n) ? n : null;
  }

  return {
    id: coerceString(o.id ?? o.model_id ?? o.modelId, ""),
    brand: coerceString(o.brand ?? o.make, ""),
    model: coerceString(o.model ?? o.model_name ?? o.modelName, ""),
    generation: coerceString(
      o.generation ?? o.gen ?? o.series ?? o.generation_code,
      ""
    ),
    years: coerceString(o.years ?? o.year ?? o.model_year ?? o.modelYear, ""),
    version: coerceString(o.version ?? o.version_tag ?? o.tag, ""),
    file: coerceString(
      o.file ?? o.file_name ?? o.fileName ?? o.weights_path ?? o.path,
      ""
    ),
    parts_catalog: coerceString(
      o.parts_catalog ??
        o.partsCatalog ??
        o.parts_catalog_uri ??
        o.catalog,
      ""
    ),
    accuracy,
    created_at: coerceString(
      o.created_at ?? o.createdAt ?? o.updated_at ?? o.updatedAt,
      ""
    ),
    status: normalizeMLModelStatus(o.status ?? o.lifecycle ?? o.state),
  };
}

/**
 * @param {unknown} raw
 */
export function normalizeMLModelList(raw) {
  const list = extractItemsArray(raw, [
    "items",
    "results",
    "models",
    "ml_models",
    "data",
  ]);
  return list.map((row) => normalizeMLModel(row)).filter((r) => r.id);
}

/**
 * @param {unknown} payload
 */
export function normalizeMLUploadPayload(payload) {
  const p =
    typeof payload === "object" && payload !== null && !Array.isArray(payload)
      ? /** @type {Record<string, unknown>} */ (payload)
      : {};

  const brand = coerceString(p.brand, "").slice(0, 160);
  const model = coerceString(p.model, "").slice(0, 160);
  const generation = coerceString(p.generation, "").slice(0, 120);
  const years = coerceString(p.years ?? p.year, "").slice(0, 48);

  const modelFile =
    p.modelFile instanceof File
      ? p.modelFile
      : p.model_file instanceof File
        ? p.model_file
        : null;
  const partsCatalogFile =
    p.partsCatalogFile instanceof File
      ? p.partsCatalogFile
      : p.parts_catalog instanceof File || p.partsCatalog instanceof File
        ? /** @type {File} */ (p.parts_catalog ?? p.partsCatalog)
        : null;

  if (
    !(modelFile instanceof File) ||
    !(partsCatalogFile instanceof File)
  ) {
    adminDevLog("ml.upload.invalid_files", {});
    throw new Error("Укажите файл модели и каталог деталей (.json)");
  }

  return {
    brand,
    model,
    generation,
    years,
    modelFile,
    partsCatalogFile,
  };
}

/**
 * multipart field names tolerated by typical backends / proxies
 *
 * @param {unknown} payload
 */
export function buildMLModelUploadFormData(payload) {
  const p = normalizeMLUploadPayload(payload);
  const form = new FormData();
  form.append("brand", p.brand);
  form.append("model", p.model);
  form.append("generation", p.generation);
  form.append("years", p.years);

  try {
    form.append("model_file", p.modelFile, p.modelFile.name);
  } catch {
    form.append("model_file", p.modelFile);
  }
  try {
    form.append(
      "parts_catalog",
      p.partsCatalogFile,
      p.partsCatalogFile.name
    );
  } catch {
    form.append("parts_catalog", p.partsCatalogFile);
  }

  return form;
}

/**
 * @param {unknown} prev
 * @param {unknown} srv
 */
export function mergeMLModel(prev, srv) {
  const a = prev ? normalizeMLModel(prev) : null;

  if (srv === null || srv === undefined) {
    return a ?? normalizeMLModel({});
  }

  /** @type {Record<string, unknown>} */
  const patch =
    typeof srv === "object" && srv !== null && !Array.isArray(srv)
      ? /** @type {Record<string, unknown>} */ (srv)
      : {};

  if (Object.keys(patch).length === 0 && a) {
    return a;
  }

  if (!a) {
    return normalizeMLModel(patch);
  }

  /** @type {Record<string, unknown>} */
  const merged = /** @type {Record<string, unknown>} */ ({
    brand: patch.brand ?? a.brand,
    model: patch.model ?? a.model,
    generation: patch.generation ?? a.generation,
    years: patch.years ?? a.years,
    version: patch.version ?? a.version,
    file: patch.file ?? a.file,
    parts_catalog: patch.parts_catalog ?? patch.partsCatalog ?? a.parts_catalog,
    accuracy: patch.accuracy !== undefined ? patch.accuracy : a.accuracy,
    created_at: patch.created_at ?? patch.createdAt ?? a.created_at,
    status:
      patch.status !== undefined && patch.status !== null
        ? patch.status
        : a.status,
    id: patch.id ?? a.id,
  });

  return normalizeMLModel(merged);
}

/* ---- Service registrations (admin queue) ---- */

/** @param {unknown} raw */
export function normalizeServiceRegistrationAdminStatus(raw) {
  const s = coerceString(raw, "").toLowerCase().replace(/\s+/g, "_");
  if (["pending", "new", "submitted", "open", "received"].includes(s)) {
    return "pending";
  }
  if (["approved", "accepted", "confirmed"].includes(s)) return "approved";
  if (["rejected", "declined", "denied", "cancelled", "canceled"].includes(s)) {
    return "rejected";
  }
  if (s) return s;
  return "pending";
}

/** @param {unknown} raw */
export function normalizeServiceRegistrationAdmin(raw) {
  const r = normalizeServiceRegistration(raw);
  return {
    ...r,
    status: normalizeServiceRegistrationAdminStatus(r.status),
  };
}

/**
 * @param {unknown} raw
 */
export function normalizeServiceRegistrationList(raw) {
  const list = extractItemsArray(raw, [
    "items",
    "results",
    "registrations",
    "service_registrations",
    "requests",
    "data",
  ]);
  return list
    .map((row) => normalizeServiceRegistrationAdmin(row))
    .filter((r) => r.id);
}

/** @param {unknown} options */
export function normalizeServiceRegistrationActionPayload(options = {}) {
  const o =
    typeof options === "object" &&
    options !== null &&
    !Array.isArray(options)
      ? /** @type {Record<string, unknown>} */ (options)
      : {};

  const reason =
    typeof o.reason === "string" ? o.reason.trim().slice(0, 4000) : "";
  return { reason };
}

/**
 * Pending queue only — blocks double-approve/race drift.
 *
 * @param {unknown} fromRaw
 */
export function assertServiceRegistrationQueueAction(fromRaw) {
  const s = normalizeServiceRegistrationAdminStatus(fromRaw);
  if (s !== "pending") {
    adminDevLog("service_reg.stale_action", {});
    throw new Error("Заявка уже обработана. Обновите список.");
  }
}

/**
 * @param {unknown} prev
 * @param {unknown} srv
 */
export function mergeServiceRegistrationAdmin(prev, srv) {
  const a = normalizeServiceRegistration(prev ?? {});
  const b = normalizeServiceRegistration(srv ?? {});
  return normalizeServiceRegistrationAdmin({
    ...a,
    ...b,
    id: coerceString(b.id, "") || a.id,
    contact_phone: coerceString(b.contact_phone, "") || a.contact_phone,
    contact_email: coerceString(b.contact_email, "") || a.contact_email,
    contact_name: coerceString(b.contact_name, "") || a.contact_name,
    rejection_reason:
      coerceString(b.rejection_reason, "").trim() !== ""
        ? b.rejection_reason
        : a.rejection_reason,
    status:
      b.status !== undefined && b.status !== null && coerceString(String(b.status), "") !== ""
        ? b.status
        : a.status,
  });
}

/* ---- Training requests (admin queue) ---- */

const ADMIN_TRAINING_STATUSES = new Set([
  "pending",
  "approved",
  "rejected",
  "completed",
  "in_progress",
]);

/** @param {unknown} raw */
export function normalizeTrainingRequestAdminStatus(raw) {
  const s = coerceString(raw, "").toLowerCase().replace(/\s+/g, "_");
  if (["pending", "new", "submitted", "open"].includes(s)) return "pending";
  if (["approved", "accepted"].includes(s)) return "approved";
  if (["rejected", "declined", "denied"].includes(s)) return "rejected";
  if (["completed", "done"].includes(s)) return "completed";
  if (["in_progress", "training", "in-training"].includes(s)) {
    return "in_progress";
  }
  return s || "pending";
}

/** @param {unknown} raw */
export function normalizeTrainingRequestAdmin(raw) {
  const r = normalizeTrainingRequest(raw);
  return {
    ...r,
    status: normalizeTrainingRequestAdminStatus(r.status),
  };
}

/** @param {unknown} raw */
export function normalizeTrainingRequestList(raw) {
  const list = extractItemsArray(raw, [
    "items",
    "results",
    "requests",
    "training_requests",
    "data",
  ]);
  return list
    .map((row) => normalizeTrainingRequestAdmin(row))
    .filter((r) => r.id);
}

/** @param {unknown} status */
export function normalizeTrainingRequestStatusPayload(status) {
  const canon = normalizeTrainingRequestAdminStatus(status);
  if (!ADMIN_TRAINING_STATUSES.has(canon)) {
    adminDevLog("training.payload.invalid_status", {});
    throw new Error("Некорректный статус заявки");
  }
  return { status: canon };
}

/** @type {Record<string, Set<string>>} */
const TRAINING_EDGES = {
  pending: new Set(["approved", "rejected"]),
  approved: new Set(["completed", "rejected", "in_progress"]),
  in_progress: new Set(["completed", "rejected"]),
  rejected: new Set(),
  completed: new Set(),
};

/**
 * Guards invalid admin transitions (race / stale FE).
 *
 * @param {unknown} fromRaw
 * @param {unknown} toRaw
 */
export function assertTrainingTransitionAllowed(fromRaw, toRaw) {
  const from = normalizeTrainingRequestAdminStatus(fromRaw);
  const to = normalizeTrainingRequestAdminStatus(toRaw);
  normalizeTrainingRequestStatusPayload(to);
  if (from === to) return;

  const allowed = TRAINING_EDGES[from];
  if (!allowed || !allowed.has(to)) {
    adminDevLog("training.transition.rejected", { fromAllowed: !!allowed });
    throw new Error("Недопустимое изменение статуса заявки");
  }
}

/**
 * @param {unknown} prev
 * @param {unknown} srv
 */
export function mergeTrainingRequestAdmin(prev, srv) {
  const a = normalizeTrainingRequest(prev ?? {});
  const b = normalizeTrainingRequest(srv ?? {});
  const y =
    coerceString(b.year, "").trim() !== ""
      ? b.year
      : coerceString(b.years, "").trim() !== ""
        ? String(b.years)
        : a.year;

  return normalizeTrainingRequestAdmin({
    ...a,
    ...b,
    id: coerceString(b.id, "") || a.id,
    year: y || a.year,
    years: y || a.year,
    description:
      coerceString(b.description, "").trim() !== ""
        ? b.description
        : a.description,
    status:
      b.status !== undefined && b.status !== null && coerceString(String(b.status), "") !== ""
        ? b.status
        : a.status,
  });
}
