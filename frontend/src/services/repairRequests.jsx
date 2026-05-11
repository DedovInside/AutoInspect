/**
 * Domain: repair requests (заявки на ремонт).
 * Production path: apiClient + normalization. Mock path: gated by env.
 */

import { apiClient } from "./apiClient";
import { normalizeApiError } from "./apiFoundation";
import { repairDevLog } from "./repairDebug";
import {
  normalizeCreateRepairPayload,
  normalizeRepairRequest,
  normalizeRepairRequestList,
  normalizeRepairService,
} from "./repairNormalize";
import {
  MOCK_REPAIR_REQUESTS_SERVICE,
  MOCK_REPAIR_REQUESTS_USER,
} from "./mockData";

export {
  mergeRepairRequest,
  normalizeCreateRepairPayload,
  normalizeRepairRequest,
  normalizeRepairRequestList,
  normalizeRepairRequestStatus,
  normalizeRepairService,
  normalizeRepairUserContact,
  unwrapRepairEnvelope,
} from "./repairNormalize";

function delay(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Prod: mocks only if `VITE_USE_MOCK_REPAIR_REQUESTS=true`.
 * Dev: mocks on unless explicit `false`.
 */
function shouldUseRepairMocks() {
  if (import.meta.env.VITE_USE_MOCK_REPAIR_REQUESTS === "true") return true;
  if (import.meta.env.VITE_USE_MOCK_REPAIR_REQUESTS === "false") return false;
  return import.meta.env.DEV;
}

/**
 * @param {AbortSignal[] | (AbortSignal | undefined)[]} signals
 */
function combineAbortSignals(...signals) {
  const valid = signals.filter((s) => s != null);
  if (valid.length === 0) return undefined;
  if (typeof AbortSignal !== "undefined" && AbortSignal.any) {
    try {
      return AbortSignal.any(valid);
    } catch {
      /* ignore */
    }
  }
  const c = new AbortController();
  const onAbort = () => c.abort();
  for (const s of valid) {
    if (s.aborted) {
      onAbort();
      break;
    }
    s.addEventListener("abort", onAbort, { once: true });
  }
  return c.signal;
}

/**
 * @param {number} ms
 */
function abortAfter(ms) {
  if (!ms || ms <= 0) {
    return { signal: undefined, clear: () => {} };
  }
  const c = new AbortController();
  const t = setTimeout(() => c.abort(), ms);
  return {
    signal: c.signal,
    clear: () => clearTimeout(t),
  };
}

/** @typedef {Awaited<ReturnType<typeof normalizeRepairRequest>>} RepairRow */

/** @type {RepairRow[]} */
let mockUserRequests = MOCK_REPAIR_REQUESTS_USER.map((r) =>
  normalizeRepairRequest(r)
);
/** @type {RepairRow[]} */
let mockServiceRequests = MOCK_REPAIR_REQUESTS_SERVICE.map((r) =>
  normalizeRepairRequest(r)
);

function normalizeListResponseShape(raw, source) {
  let next_cursor;
  /** @type {unknown} */
  let body = raw;

  if (
    raw &&
    typeof raw === "object" &&
    !Array.isArray(raw)
  ) {
    /** @type {Record<string, unknown>} */
    const o = /** @type {Record<string, unknown>} */ (raw);
    next_cursor = o.next_cursor ?? o.nextCursor ?? undefined;
    if (next_cursor !== undefined && import.meta.env.DEV) {
      repairDevLog("list.pagination_hint", { source: source, hasNext: true });
    }
    body = o.items ?? o.results ?? o.requests ?? raw;
    if (
      body &&
      typeof body === "object" &&
      !Array.isArray(body)
    ) {
      /** @type {Record<string, unknown>} */
      const inner = /** @type {Record<string, unknown>} */ (body);
      body = inner.items ?? inner.data ?? inner.results ?? raw;
    }
  }

  const items = normalizeRepairRequestList(body);
  if ((!Array.isArray(raw) || !raw.every((x) => x && typeof x === "object")) && items.length === 0 && raw != null && import.meta.env.DEV) {
    repairDevLog("list.empty_or_invalid_after_normalize", {
      source: source,
      kind:
        typeof raw === "object" && raw !== null && !Array.isArray(raw)
          ? "object"
          : Array.isArray(raw)
          ? "array"
          : typeof raw,
    });
  }
  return { items, next_cursor };
}

function filterByStatus(items, status) {
  if (!status || status === "all") return [...items];
  return items.filter((r) => r.status === status);
}

/**
 * Normalize user-facing mutation errors without leaking payloads.
 */
function mapRepairMutationError(stage, err) {
  const n = normalizeApiError(err);

  repairDevLog(`${stage}_fail`, {
    aborted: n.code === "aborted",
    status: n.status,
    code: n.code,
  });

  if (n.code === "aborted") {
    throw err instanceof Error ? err : new Error(n.message || "Запрос отменён");
  }

  if (n.status === 0) {
    return new Error(n.message || "Не удалось связаться с сервером");
  }
  if (n.status === 409) {
    return new Error(
      "Заявка уже изменена или обработана на сервере. Обновите список."
    );
  }
  if (typeof n.message === "string" && n.message.trim()) {
    return new Error(n.message);
  }
  return new Error(`Ошибка: ${stage}`);
}

/** Dedupe simultaneous creates for same analysis/service (same-flight coalescing) */
const inFlightCreates = new Map();
/** Dedupe simultaneous calls per operation + id (accept/reject tracked separately). */
const inFlightMutations = new Map();

/** Optional display override from UI (never sent unless backend adds fields) */
function pickServiceDisplay(payload) {
  const p = payload ?? {};
  const block = p.service ?? p.service_display ?? null;
  if (!block || typeof block !== "object") return normalizeRepairService(null);
  return normalizeRepairService(block);
}

function applyCreateFallbacks(norm, defaults) {
  const st = typeof norm.status === "string" ? norm.status : "pending";
  const note =
    norm.note && coerceString(norm.note, "") ? norm.note : coerceNoteDefault(st);
  const useFallbackSvc =
    !norm.service ||
    !norm.service.name ||
    coerceString(norm.service.name, "") === "" ||
    norm.service.name === "—";

  return {
    ...norm,
    analysis_id: norm.analysis_id || defaults.analysis_id || "",
    car_brand: norm.car_brand || defaults.car_brand || "—",
    damage_summary:
      norm.damage_summary || defaults.damage_summary || "—",
    note: note || coerceNoteDefault("pending"),
    service: useFallbackSvc
      ? defaults.service_fallback || norm.service || normalizeRepairService(null)
      : norm.service,
  };
}

function coerceNoteDefault(status) {
  if (status === "accepted")
    return "Заявка принята — свяжитесь с клиентом по указанным контактам.";
  if (status === "rejected")
    return "Заявка отклонена.";
  return "Заявка отправлена, ожидаем ответа автосервиса.";
}

/** @returns {RepairRow[]} */
async function mockListMine({ status } = {}) {
  await delay(150);
  return filterByStatus(mockUserRequests, status);
}

/** @returns {RepairRow[]} */
async function mockListIncoming({ status } = {}) {
  await delay(150);
  return filterByStatus(mockServiceRequests, status);
}

async function mockCreateRepairRequest(payload) {
  await delay(300);
  const n = normalizeCreateRepairPayload(payload);
  const svcFromUi = pickServiceDisplay(payload);

  /** @type {RepairRow} */
  const row = normalizeRepairRequest(
    {
      id: `r-${Date.now()}`,
      created_at: new Date().toISOString(),
      car_brand: n.car_brand ?? "—",
      analysis_id: n.analysis_id,
      service:
        svcFromUi?.name !== "—"
          ? svcFromUi
          : normalizeRepairService(null),
      status: "pending",
      note: coerceNoteDefault("pending"),
      damage_summary: n.damage_summary ?? "—",
    },
    {}
  );

  mockUserRequests = [row, ...mockUserRequests];
  return row;
}

async function mockAcceptRepairRequest(id) {
  await delay(150);
  const sid = String(id);
  mockServiceRequests = mockServiceRequests.map((r) =>
    r.id === sid ? { ...r, status: "accepted", note: coerceNoteDefault("accepted") } : r
  );
  let found = mockServiceRequests.find((r) => r.id === sid);
  if (!found) {
    found = normalizeRepairRequest({ id: sid, status: "accepted" });
  }
  return found;
}

async function mockRejectRepairRequest(id, { reason }) {
  await delay(150);
  const sid = String(id);
  mockServiceRequests = mockServiceRequests.map((r) =>
    r.id === sid
      ? {
          ...r,
          status: "rejected",
          rejection_reason: reason || "",
          note: coerceNoteDefault("rejected"),
        }
      : r
  );
  let found = mockServiceRequests.find((r) => r.id === sid);
  if (!found) {
    found = normalizeRepairRequest({
      id: sid,
      status: "rejected",
      rejection_reason: reason || "",
    });
  }
  return found;
}

/**
 * GET /v1/repair-requests/me?status=&cursor=&limit=
 * @param {{ status?: string, cursor?: string, limit?: number, signal?: AbortSignal, timeoutMs?: number }} [options]
 * @returns {Promise<RepairRow[]>}
 */
export async function listMyRepairRequests(options = {}) {
  const { status, cursor, limit, signal: userSignal, timeoutMs = 25_000 } =
    options;

  if (shouldUseRepairMocks()) {
    repairDevLog("list.me.mock", {});
    const rows = await mockListMine({ status });
    repairDevLog("list.me.done", { count: rows.length, mock: true });
    return rows;
  }

  const { signal: timeoutSig, clear } = abortAfter(timeoutMs);
  const combined = combineAbortSignals(userSignal, timeoutSig);

  try {
    repairDevLog("list.me.start", { status: status || "all" });
    /** @type {unknown} */
    const raw = await apiClient.get("/v1/repair-requests/me", {
      signal: combined,
      auth: true,
      query: { status: status || "", cursor: cursor || "", limit: limit ?? "" },
    });
    const { items } = normalizeListResponseShape(raw, "me");
    repairDevLog("list.me.done", { count: items.length, mock: false });
    return items;
  } catch (e) {
    repairDevLog("list.me.fail", { status: normalizeApiError(e).status });
    throw e;
  } finally {
    clear();
  }
}

/**
 * GET /v1/repair-requests/incoming?...
 * @param {{ status?: string, cursor?: string, limit?: number, signal?: AbortSignal, timeoutMs?: number }} [options]
 * @returns {Promise<RepairRow[]>}
 */
export async function listIncomingRepairRequests(options = {}) {
  const { status, cursor, limit, signal: userSignal, timeoutMs = 25_000 } =
    options;

  if (shouldUseRepairMocks()) {
    repairDevLog("list.incoming.mock", {});
    const rows = await mockListIncoming({ status });
    repairDevLog("list.incoming.done", { count: rows.length, mock: true });
    return rows;
  }

  const { signal: timeoutSig, clear } = abortAfter(timeoutMs);
  const combined = combineAbortSignals(userSignal, timeoutSig);

  try {
    repairDevLog("list.incoming.start", { status: status || "all" });
    /** @type {unknown} */
    const raw = await apiClient.get("/v1/repair-requests/incoming", {
      signal: combined,
      auth: true,
      query: { status: status || "", cursor: cursor || "", limit: limit ?? "" },
    });
    const { items } = normalizeListResponseShape(raw, "incoming");
    repairDevLog("list.incoming.done", { count: items.length, mock: false });
    return items;
  } catch (e) {
    repairDevLog("list.incoming.fail", { status: normalizeApiError(e).status });
    throw e;
  } finally {
    clear();
  }
}

async function createRepairRequestCore(payload, options = {}) {
  const { signal: userSignal, timeoutMs = 30_000 } = options ?? {};

  let n;
  try {
    n = normalizeCreateRepairPayload(payload);
  } catch (e) {
    const msg = e?.message ?? "invalid_payload";
    repairDevLog("create.invalid_payload", { stage: msg });
    throw e instanceof Error
      ? e
      : new Error(typeof msg === "string" ? msg : "Ошибка заявки");
  }

  const defaults = {
    analysis_id: n.analysis_id,
    car_brand: n.car_brand || "",
    damage_summary: n.damage_summary || "",
    service_fallback: pickServiceDisplay(payload),
  };

  if (shouldUseRepairMocks()) {
    repairDevLog("create.start", { mock: true });
    const row = applyCreateFallbacks(
      await mockCreateRepairRequest(payload),
      defaults
    );
    repairDevLog("create.success", {
      mock: true,
      hasId: !!row?.id,
    });
    return row;
  }

  const { signal: timeoutSig, clear } = abortAfter(timeoutMs);
  const combined = combineAbortSignals(userSignal, timeoutSig);

  try {
    repairDevLog("create.start", {
      mock: false,
      hasAnalysis: !!n.analysis_id,
      hasService: !!n.service_id,
    });
    const apiBody = { analysis_id: n.analysis_id, service_id: n.service_id };

    /** @type {unknown} */
    let raw = await apiClient.post("/v1/repair-requests", apiBody, {
      signal: combined,
      auth: true,
    });

    if (raw === null || (typeof raw === "object" && !Array.isArray(raw))) {
      const obj = raw && typeof raw === "object" ? raw : {};
      const empty =
        raw === null || Object.keys(/** @type {object} */ (obj)).length === 0;
      if (empty && import.meta.env.DEV) {
        repairDevLog("create.partial_or_empty_body", {});
      }
    }

    let norm = normalizeRepairRequest(raw ?? {}, {
      defaults: {
        analysis_id: n.analysis_id,
        car_brand: defaults.car_brand,
        damage_summary: defaults.damage_summary,
      },
    });

    if (!norm.id) {
      repairDevLog("create.fallback_id", {});
      norm = normalizeRepairRequest(
        { ...ensureObject(raw || {}), id: `pending-${Date.now()}` },
        {
          defaults: {
            analysis_id: n.analysis_id,
            car_brand: defaults.car_brand,
            damage_summary: defaults.damage_summary,
          },
        }
      );
    }

    norm = applyCreateFallbacks(norm, defaults);
    repairDevLog("create.success", {
      mock: false,
      hasId: !!norm?.id,
    });
    return norm;
  } catch (e) {
    repairDevLog("create.fail", {
      aborted: normalizeApiError(e).code === "aborted",
      status: normalizeApiError(e).status,
      code: normalizeApiError(e).code,
    });
    throw mapRepairMutationError("createRepairRequest", e);
  } finally {
    clear();
  }
}

function ensureObject(x) {
  return x && typeof x === "object" && !Array.isArray(x) ? x : {};
}

/**
 * POST /v1/repair-requests  (BODY: analysis_id, service_id)
 * Coalesces parallel duplicate submits for identical analysis/service.
 * Extra keys on payload (car_brand, damage_summary, service object) enrich mocks/partials — not sent unless API evolves.
 *
 * @param {Record<string, unknown>} payload
 * @param {{ signal?: AbortSignal, timeoutMs?: number }} [options]
 */
export async function createRepairRequest(payload, options = {}) {
  const n = normalizeCreateRepairPayload(payload);
  const key = `${n.analysis_id}:${n.service_id}`;
  if (inFlightCreates.has(key)) {
    repairDevLog("create.dedupe_reuse_inflight", {
      reuse: true,
    });
    return inFlightCreates.get(key);
  }
  const promise = createRepairRequestCore(payload, options).finally(() => {
    inFlightCreates.delete(key);
  });
  inFlightCreates.set(key, promise);
  return promise;
}

/**
 * @param {string} id
 * @param {{ signal?: AbortSignal, timeoutMs?: number }} [options]
 */
export async function acceptRepairRequest(id, options = {}) {
  const sid = String(id ?? "");
  if (!sid) {
    return normalizeRepairRequest({});
  }

  const opKey = `accept:${sid}`;
  if (inFlightMutations.has(opKey)) {
    repairDevLog("mutation.dedupe", { action: "accept" });
    return inFlightMutations.get(opKey);
  }

  const run = async () => {
    if (shouldUseRepairMocks()) {
      repairDevLog("accept.start", { mock: true });
      const updated = normalizeRepairRequest(await mockAcceptRepairRequest(sid));
      repairDevLog("accept.success", { mock: true });
      return updated;
    }

    const { signal: userSignal, timeoutMs = 25_000 } = options ?? {};
    const { signal: timeoutSig, clear } = abortAfter(timeoutMs);
    const combined = combineAbortSignals(userSignal, timeoutSig);

    try {
      repairDevLog("accept.start", { mock: false });
      /** @type {unknown} */
      const raw = await apiClient.post(
        `/v1/repair-requests/${encodeURIComponent(sid)}/accept`,
        {},
        { signal: combined, auth: true }
      );

      let norm = normalizeRepairRequest(raw ?? { id: sid, status: "accepted" });
      if (norm.status !== "accepted") {
        repairDevLog("accept.normalize_status_correction", {});
        norm = { ...norm, status: "accepted", id: norm.id || sid };
      }
      repairDevLog("accept.success", { mock: false });
      return norm;
    } catch (e) {
      throw mapRepairMutationError("accept", e);
    } finally {
      clear();
    }
  };

  const p = run().finally(() => {
    inFlightMutations.delete(opKey);
  });
  inFlightMutations.set(opKey, p);
  return p;
}

/**
 * @param {string} id
 * @param {{ reason?: string, signal?: AbortSignal, timeoutMs?: number }} [options]
 */
export async function rejectRepairRequest(id, options = {}) {
  const sid = String(id ?? "");
  if (!sid) {
    return normalizeRepairRequest({ status: "rejected" });
  }

  const opKey = `reject:${sid}`;
  if (inFlightMutations.has(opKey)) {
    repairDevLog("mutation.dedupe", { action: "reject" });
    return inFlightMutations.get(opKey);
  }

  const reason = typeof options.reason === "string" ? options.reason : "";

  const run = async () => {
    if (shouldUseRepairMocks()) {
      repairDevLog("reject.start", { mock: true });
      const updated = normalizeRepairRequest(
        await mockRejectRepairRequest(sid, { reason })
      );
      repairDevLog("reject.success", { mock: true });
      return updated;
    }

    const { signal: userSignal, timeoutMs = 25_000 } = options ?? {};
    const { signal: timeoutSig, clear } = abortAfter(timeoutMs);
    const combined = combineAbortSignals(userSignal, timeoutSig);

    try {
      repairDevLog("reject.start", { mock: false });
      /** @type {unknown} */
      const raw = await apiClient.post(
        `/v1/repair-requests/${encodeURIComponent(sid)}/reject`,
        { reason },
        { signal: combined, auth: true }
      );

      let norm = normalizeRepairRequest(
        raw ?? { id: sid, status: "rejected", rejection_reason: reason }
      );
      if (norm.status !== "rejected") {
        repairDevLog("reject.normalize_status_correction", {});
        norm = {
          ...norm,
          status: "rejected",
          rejection_reason:
            norm.rejection_reason ||
            coerceString(reason, ""),
          id: norm.id || sid,
        };
      }
      repairDevLog("reject.success", { mock: false });
      return norm;
    } catch (e) {
      throw mapRepairMutationError("reject", e);
    } finally {
      clear();
    }
  };

  const p = run().finally(() => {
    inFlightMutations.delete(opKey);
  });
  inFlightMutations.set(opKey, p);
  return p;
}

function coerceString(v, fb) {
  if (v == null || v === "") return fb;
  const s = String(v).trim();
  return s || fb;
}
