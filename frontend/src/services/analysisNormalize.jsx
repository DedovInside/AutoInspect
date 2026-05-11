/**
 * Canonical analysis shapes for UI + backend variance (nested data, camelCase).
 * Primary contract matches existing mock/UI: snake_case detail fields.
 */

import { analysisDevLog } from "./analysisDebug";

/** @typedef {'pending'|'queued'|'processing'|'done'|'failed'} CanonicalAnalysisStatus */

const TERMINAL_SUCCESS = new Set(["done"]);
const TERMINAL_FAILURE = new Set(["failed", "error"]);
const TERMINAL = new Set(["done", "failed", "error"]);

/**
 * Unwrap `{ data }` etc.
 * @param {unknown} raw
 */
export function unwrapAnalysisEnvelope(raw) {
  if (raw === null || raw === undefined) return raw;
  if (typeof raw !== "object" || Array.isArray(raw)) return raw;

  /** @type {Record<string, unknown>} */
  const top = /** @type {Record<string, unknown>} */ (raw);

  if (
    top.data !== undefined &&
    typeof top.data === "object" &&
    !Array.isArray(top.data) &&
    (/** @type {Record<string, unknown>} */ (top.data)).analysis !== undefined
  ) {
    return /** @type {Record<string, unknown>} */ (top.data).analysis;
  }

  if (
    top.data !== undefined &&
    typeof top.data === "object" &&
    !Array.isArray(top.data)
  ) {
    return top.data;
  }

  if (top.analysis !== undefined && typeof top.analysis === "object") {
    return top.analysis;
  }

  return top;
}

/**
 * Normalize status string → canonical lifecycle tokens.
 * @param {unknown} raw
 * @returns {CanonicalAnalysisStatus}
 */
export function normalizeAnalysisStatus(raw) {
  const s =
    raw === null || raw === undefined
      ? ""
      : String(raw).trim().toLowerCase().replace(/\s+/g, "_");

  switch (s) {
    case "pending":
    case "waiting":
      return "pending";
    case "queued":
    case "queue":
    case "enqueued":
      return "queued";
    case "processing":
    case "running":
    case "in_progress":
    case "in-progress":
      return "processing";
    case "completed":
    case "complete":
    case "done":
    case "success":
    case "succeeded":
      return "done";
    case "failed":
    case "failure":
    case "error":
    case "cancelled":
    case "canceled":
      return "failed";
    default:
      return "processing";
  }
}

/**
 * @param {CanonicalAnalysisStatus | string} status
 */
export function isTerminalAnalysisStatus(status) {
  return TERMINAL.has(String(status));
}

/**
 * Success terminal (shows result UI).
 * @param {CanonicalAnalysisStatus | string} status
 */
export function isSuccessTerminalAnalysisStatus(status) {
  return TERMINAL_SUCCESS.has(String(status));
}

/**
 * Failure terminal (stop polling).
 * @param {CanonicalAnalysisStatus | string} status
 */
export function isFailedTerminalAnalysisStatus(status) {
  return TERMINAL_FAILURE.has(String(status));
}

/** @param {unknown} raw */
export function coerceString(raw, fallback = "") {
  if (raw === null || raw === undefined) return fallback;
  const s = String(raw).trim();
  return s || fallback;
}

/** @param {unknown} src */
function normalizeDamages(src) {
  const env = unwrapAnalysisEnvelope(src);
  const list =
    Array.isArray(src)
      ? src
      : env &&
          typeof env === "object" &&
          !Array.isArray(env) &&
          Array.isArray(
            /** @type {{ damages?: unknown }} */ (env).damages
          )
        ? /** @type {{ damages: unknown[] }} */ (env).damages
        : env &&
            typeof env === "object" &&
            !Array.isArray(env) &&
            Array.isArray(
              /** @type {{ detections?: unknown }} */ (env).detections
            )
          ? /** @type {{ detections: unknown[] }} */ (env).detections
          : [];

  if (!Array.isArray(list)) return [];

  return list
    .map((d, index) => {
      if (!d || typeof d !== "object") {
        return { part: `Повреждение ${index + 1}`, severity: "Не указано" };
      }
      const o = /** @type {Record<string, unknown>} */ (d);
      const part = coerceString(
        o.part ?? o.name ?? o.area ?? o.region ?? o.label,
        "Не указано"
      );
      const severity = coerceString(
        o.severity ?? o.level ?? o.damage_severity ?? o.damageSeverity,
        "Не указано"
      );
      return { part, severity };
    })
    .filter(Boolean);
}

/** @param {unknown} src */
function normalizeServices(src) {
  const env = unwrapAnalysisEnvelope(src);

  /** @type {unknown} */
  let list =
    Array.isArray(src)
      ? src
      : env &&
          typeof env === "object" &&
          !Array.isArray(env) &&
          Array.isArray(
            /** @type {{ services?: unknown }} */ (env).services
          )
        ? /** @type {{ services: unknown[] }} */ (env).services
        : env &&
            typeof env === "object" &&
            !Array.isArray(env) &&
            Array.isArray(
              /** @type {{ recommendations?: unknown }} */ (env).recommendations
            )
          ? /** @type {{ recommendations: unknown[] }} */ (
              env
            ).recommendations
          : [];

  if (!Array.isArray(list)) return [];

  return list
    .map((s, index) => {
      if (!s || typeof s !== "object") {
        return {
          id: `svc-${index}`,
          name: "Сервис",
          phone: "—",
          address: "—",
          description: "",
        };
      }
      const o = /** @type {Record<string, unknown>} */ (s);
      const id = coerceString(
        o.id ?? o.service_id ?? o.serviceId,
        `svc-${index}`
      );
      const name = coerceString(o.name ?? o.title, "Сервис");
      const phone = coerceString(
        o.phone ?? o.phone_number ?? o.phoneNumber,
        "—"
      );
      const address = coerceString(
        o.address ?? o.location ?? o.full_address ?? o.fullAddress,
        "—"
      );
      const description = coerceString(
        o.description ?? o.desc ?? o.summary,
        ""
      );
      return { id, name, phone, address, description };
    })
    .filter(Boolean);
}

/**
 * Full analysis document for result screen.
 * @param {unknown} raw
 * @param {{ routeId?: string }} [ctx]
 */
export function normalizeAnalysisResponse(raw, ctx = {}) {
  const envelope = unwrapAnalysisEnvelope(raw);
  if (
    envelope === null ||
    envelope === undefined ||
    typeof envelope !== "object" ||
    Array.isArray(envelope)
  ) {
    analysisDevLog("normalize.blank_envelope", { routeId: ctx.routeId ?? null });
    const rid = ctx.routeId ? String(ctx.routeId) : "";
    return {
      id: rid,
      analysis_id: rid,
      status: "failed",
      brand: "—",
      created_at: new Date().toISOString(),
      overall_severity: "—",
      damages: [],
      services: [],
    };
  }

  /** @type {Record<string, unknown>} */
  const o = /** @type {Record<string, unknown>} */ (envelope);

  const canonicalStatus = normalizeAnalysisStatus(
    o.status ?? o.state ?? o.phase ?? o.analysis_status ?? o.analysisStatus
  );

  const id = coerceString(
    o.analysis_id ??
      o.analysisId ??
      o.id ??
      (ctx.routeId != null ? String(ctx.routeId) : ""),
    ctx.routeId != null ? String(ctx.routeId) : "unknown"
  );

  const brand = coerceString(
    o.brand ?? o.vehicle_brand ?? o.vehicleBrand,
    "—"
  );

  const created_at = coerceString(
    o.created_at ?? o.createdAt ?? o.completed_at ?? o.completedAt,
    new Date().toISOString()
  );

  const overall_severity = coerceString(
    o.overall_severity ??
      o.overallSeverity ??
      o.severity_summary ??
      o.damage_level,
    "Не указано"
  );

  const damages = normalizeDamages(
    o.damages ?? o.detections ?? o.damage_items ?? o.damageItems
  );
  const services = normalizeServices(
    o.services ?? o.service_recommendations ?? o.serviceRecommendations
  );

  return {
    id,
    analysis_id: id,
    status: canonicalStatus,
    brand,
    created_at,
    overall_severity,
    damages,
    services,
  };
}

/**
 * History row shape.
 * @param {unknown} raw
 */
export function normalizeAnalysisListItem(raw) {
  const row = unwrapAnalysisEnvelope(raw);
  if (!row || typeof row !== "object" || Array.isArray(row)) {
    return {
      id: "",
      created_at: new Date().toISOString(),
      brand: "—",
      status: "processing",
      overall_severity: "—",
      damages_count: 0,
    };
  }

  /** @type {Record<string, unknown>} */
  const o = /** @type {Record<string, unknown>} */ (row);

  const id = coerceString(o.analysis_id ?? o.analysisId ?? o.id, "");
  const created_at = coerceString(
    o.created_at ?? o.createdAt,
    new Date().toISOString()
  );
  const brand = coerceString(o.brand ?? o.vehicle_brand ?? o.vehicleBrand, "—");
  const status = normalizeAnalysisStatus(
    o.status ?? o.state ?? o.phase ?? "processing"
  );
  const overall_severity = coerceString(
    o.overall_severity ?? o.overallSeverity,
    "—"
  );

  let damages_count = 0;
  if (typeof o.damages_count === "number" && Number.isFinite(o.damages_count)) {
    damages_count = o.damages_count;
  } else if (
    typeof o.damagesCount === "number" &&
    Number.isFinite(o.damagesCount)
  ) {
    damages_count = o.damagesCount;
  } else if (Array.isArray(o.damages)) {
    damages_count = o.damages.length;
  } else if (Array.isArray(o.detections)) {
    damages_count = o.detections.length;
  }

  return {
    id,
    created_at,
    brand,
    status,
    overall_severity,
    damages_count,
  };
}

/**
 * Normalize list endpoint body → array of history items.
 * @param {unknown} raw
 */
export function normalizeAnalysisList(raw) {
  if (raw === null || raw === undefined) return [];

  let list = null;

  if (Array.isArray(raw)) {
    list = raw;
  } else if (typeof raw === "object") {
    /** @type {Record<string, unknown>} */
    const o = /** @type {Record<string, unknown>} */ (raw);
    const inner =
      o.items ??
      o.results ??
      o.analyses ??
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
    .map((item) => normalizeAnalysisListItem(item))
    .filter((row) => row.id);
}

/**
 * POST /v1/analyses response → { analysis_id }.
 * @param {unknown} raw
 */
export function normalizeUploadResponse(raw) {
  const o = unwrapAnalysisEnvelope(raw);
  if (!o || typeof o !== "object" || Array.isArray(o)) {
    throw new Error("Пустой ответ при создании анализа");
  }

  /** @type {Record<string, unknown>} */
  const top = /** @type {Record<string, unknown>} */ (o);

  const id = coerceString(
    top.analysis_id ?? top.analysisId ?? top.id ?? top.job_id ?? top.jobId,
    ""
  );

  if (!id) {
    throw new Error("Сервер не вернул идентификатор анализа");
  }

  return { analysis_id: id };
}
