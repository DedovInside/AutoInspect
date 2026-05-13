/**
 * Domain: requests to be granted the SERVICE role (USER home → ADMIN queue).
 */

import { apiClient } from "./apiClient";
import { normalizeApiError } from "./apiFoundation";
import { homeFormsDevLog } from "./homeFormsDebug";
import {
  assertServiceRegistrationQueueAction,
  mergeServiceRegistrationAdmin,
  normalizeServiceRegistrationActionPayload,
  normalizeServiceRegistrationAdmin,
  normalizeServiceRegistrationAdminStatus,
  normalizeServiceRegistrationList,
  rethrowAbortOrWrapAdminError,
} from "./adminNormalize";
import {
  mergeServiceRegistrationResponse,
  normalizeServiceRegistration,
  normalizeServiceRegistrationPayload,
  serviceRegistrationPayloadToApiBody,
} from "./homeFormsNormalize";
import { MOCK_SERVICE_REQUESTS } from "./mockData";

export {
  mergeServiceRegistrationResponse,
  normalizeServiceRegistration,
  normalizeServiceRegistrationPayload,
  serviceRegistrationPayloadToApiBody,
} from "./homeFormsNormalize";

export {
  assertServiceRegistrationQueueAction,
  mergeServiceRegistrationAdmin,
  normalizeServiceRegistrationActionPayload,
  normalizeServiceRegistrationAdmin,
  normalizeServiceRegistrationList,
  rethrowAbortOrWrapAdminError,
} from "./adminNormalize";

function delay(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function shouldUseServiceRegistrationMocks() {
  if (import.meta.env.VITE_USE_MOCK_SERVICE_REGISTRATIONS === "true") {
    return true;
  }
  if (import.meta.env.VITE_USE_MOCK_SERVICE_REGISTRATIONS === "false") {
    return false;
  }
  return import.meta.env.DEV;
}

/** @param {AbortSignal[] | (AbortSignal | undefined)[]} signals */
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

/** @param {number} ms */
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

function throwSubmitRegistrationError(err) {
  const n = normalizeApiError(err);
  homeFormsDevLog(
    "registration",
    "submit.error",
    { aborted: n.code === "aborted", status: n.status, code: n.code }
  );
  if (n.code === "aborted") {
    throw err instanceof Error ? err : new Error(n.message || "Запрос отменён");
  }
  if (n.status === 422 || n.status === 400) {
    throw new Error(n.message || "Проверьте поля заявки");
  }
  if (n.status === 0) {
    throw new Error(n.message || "Не удалось связаться с сервером");
  }
  throw new Error(
    coerceMsg(n.message) || "Не удалось отправить заявку на регистрацию"
  );
}

function coerceMsg(v) {
  return typeof v === "string" && v.trim() ? v.trim() : "";
}

/** @type {Awaited<ReturnType<typeof normalizeServiceRegistrationAdmin>>[]} */
let registrations = MOCK_SERVICE_REQUESTS.map((row) =>
  normalizeServiceRegistrationAdmin(row)
);

/** @type {Map<string, Promise<unknown>>} */
const inflightApprove = new Map();
/** @type {Map<string, Promise<unknown>>} */
const inflightReject = new Map();

/**
 * @param {"approve" | "reject"} kind
 * @param {string} sid
 * @param {() => Promise<unknown>} fn
 */
function dedupeRegistrationAction(kind, sid, fn) {
  const key = `${kind}:${sid}`;
  const map = kind === "approve" ? inflightApprove : inflightReject;
  if (map.has(key)) return map.get(key);
  const p = Promise.resolve(fn()).finally(() => {
    map.delete(key);
  });
  map.set(key, p);
  return p;
}

/** Serialize overlapping submits from any caller */
let submitTail = Promise.resolve();

/**
 * @param {() => Promise<unknown>} fn
 */
function enqueueSubmit(fn) {
  const run = async () => {
    await submitTail.catch(() => {});
    return fn();
  };
  const p = run();
  submitTail = p.catch(() => {});
  return p;
}

/**
 * POST /v1/car-service-applications
 * @param {unknown} payload
 * @param {{ signal?: AbortSignal, timeoutMs?: number }} [options]
 */
export async function submitServiceRegistration(payload, options = {}) {
  const cleanedPreview = normalizeServiceRegistrationPayload(payload);
  homeFormsDevLog("registration", "submit.enqueue", {
    mock: shouldUseServiceRegistrationMocks(),
    hasOrgLen: cleanedPreview.organization.length > 0,
  });

  return enqueueSubmit(async () => {
    const cleaned = normalizeServiceRegistrationPayload(payload);

    if (shouldUseServiceRegistrationMocks()) {
      await delay(260);
      const created = mergeServiceRegistrationResponse(cleaned, {
        id: `sr-${Date.now()}`,
        submitted_at: new Date().toISOString(),
        status: "pending",
        contact_phone: cleaned.phone,
        organization: cleaned.organization,
        address: cleaned.address,
        city: cleaned.city,
        description: cleaned.description,
      });
      registrations = [
        normalizeServiceRegistrationAdmin(created),
        ...registrations,
      ];
      homeFormsDevLog("registration", "submit.success", { mock: true });
      return created;
    }

    const { signal: userSignal, timeoutMs = 35_000 } = options ?? {};
    const { signal: timeoutSig, clear } = abortAfter(timeoutMs);
    const combined = combineAbortSignals(userSignal, timeoutSig);

    try {
      const body = serviceRegistrationPayloadToApiBody(cleaned);
      /** @type {unknown} */
      const raw = await apiClient.post("/v1/car-service-applications", body, {
        signal: combined,
        auth: true,
      });

      if (
        raw === null ||
        (typeof raw === "object" &&
          !Array.isArray(raw) &&
          Object.keys(/** @type {object} */ (raw ?? {})).length === 0)
      ) {
        homeFormsDevLog("registration", "submit.partial_body", {});
      }

      const merged = mergeServiceRegistrationResponse(cleaned, raw ?? {});
      homeFormsDevLog("registration", "submit.success", { mock: false });
      return merged;
    } catch (e) {
      throwSubmitRegistrationError(e);
    } finally {
      clear();
    }
  });
}

/**
 * GET /v1/admin/car-service-applications?status=
 * @param {{ status?: string, signal?: AbortSignal, timeoutMs?: number }} [options]
 */
export async function listServiceRegistrations(options = {}) {
  const { status, signal: userSignal, timeoutMs = 25_000 } = options;

  if (shouldUseServiceRegistrationMocks()) {
    await delay(120);
    let rows = registrations.map((r) => normalizeServiceRegistrationAdmin(r));
    if (status && status !== "all") {
      rows = rows.filter(
        (r) =>
          normalizeServiceRegistrationAdminStatus(r.status) ===
          normalizeServiceRegistrationAdminStatus(status)
      );
    }
    return rows;
  }

  const { signal: timeoutSig, clear } = abortAfter(timeoutMs);
  const combined = combineAbortSignals(userSignal, timeoutSig);

  try {
    /** @type {unknown} */
    const raw = await apiClient.get("/v1/admin/car-service-applications", {
      signal: combined,
      auth: true,
      query: { status: status ?? "" },
    });
    return normalizeServiceRegistrationList(raw);
  } catch (e) {
    rethrowAbortOrWrapAdminError(e);
  } finally {
    clear();
  }
}

/**
 * GET /v1/car-service-applications?offset=&limit=
 * @param {{ signal?: AbortSignal, timeoutMs?: number }} [options]
 */
export async function listMyServiceRegistrations(options = {}) {
  const { signal: userSignal, timeoutMs = 25_000 } = options ?? {};

  if (shouldUseServiceRegistrationMocks()) {
    await delay(120);
    return registrations.map((r) => normalizeServiceRegistration(r));
  }

  const { signal: timeoutSig, clear } = abortAfter(timeoutMs);
  const combined = combineAbortSignals(userSignal, timeoutSig);

  try {
    /** @type {unknown} */
    const raw = await apiClient.get("/v1/car-service-applications", {
      signal: combined,
      auth: true,
      query: { limit: 20, offset: 0 },
    });
    return normalizeServiceRegistrationList(raw);
  } catch (e) {
    const n = normalizeApiError(e);
    if (n.code === "aborted") {
      throw e instanceof Error ? e : new Error(n.message || "Запрос отменён");
    }
    throw new Error(n.message || "Не удалось загрузить заявки на регистрацию");
  } finally {
    clear();
  }
}

function assertServiceRegFromMatchesUi(snapshot, fromOpt) {
  if (snapshot == null) return;
  if (fromOpt == null || coerceMsg(String(fromOpt)) === "") return;
  const cur = normalizeServiceRegistrationAdminStatus(
    /** @type {{ status?: string }} */ (snapshot).status
  );
  const from = normalizeServiceRegistrationAdminStatus(fromOpt);
  if (cur !== from) {
    throw new Error("Заявка уже обновлена. Обновите список.");
  }
}

async function approveServiceRegistrationImpl(id, options = {}) {
  const sid = String(id ?? "");
  const snapRowRaw =
    typeof options.snapshot === "object" &&
    options.snapshot !== null &&
    !Array.isArray(options.snapshot)
      ? options.snapshot
      : shouldUseServiceRegistrationMocks()
        ? registrations.find((r) => r.id === sid)
        : undefined;

  if (shouldUseServiceRegistrationMocks()) {
    await delay(150);
    if (!snapRowRaw || typeof snapRowRaw !== "object") {
      throw new Error("Заявка не найдена.");
    }
    const snap = normalizeServiceRegistrationAdmin(snapRowRaw);
    assertServiceRegFromMatchesUi(snap, options.from);
    assertServiceRegistrationQueueAction(snap.status);
    registrations = registrations.map((r) =>
      r.id === sid ? mergeServiceRegistrationAdmin(r, { status: "approved" }) : r
    );
    return registrations.find((r) => r.id === sid);
  }

  if (snapRowRaw && typeof snapRowRaw === "object") {
    const snap = normalizeServiceRegistrationAdmin(snapRowRaw);
    assertServiceRegFromMatchesUi(snap, options.from);
    assertServiceRegistrationQueueAction(snap.status);
  }

  const { signal: userSignal, timeoutMs = 25_000 } = options ?? {};
  const { signal: timeoutSig, clear } = abortAfter(timeoutMs);
  const combined = combineAbortSignals(userSignal, timeoutSig);

  try {
    /** @type {unknown} */
    const raw = await apiClient.patch(
      `/v1/admin/car-service-applications/${encodeURIComponent(sid)}/approve`,
      {},
      { signal: combined, auth: true }
    );
    return mergeServiceRegistrationAdmin(
      snapRowRaw && typeof snapRowRaw === "object"
        ? snapRowRaw
        : { id: sid },
      raw ?? { id: sid, status: "approved" }
    );
  } catch (e) {
    rethrowAbortOrWrapAdminError(e);
  } finally {
    clear();
  }
}

/**
 * PATCH /v1/admin/car-service-applications/:id/approve
 * @param {string} id
 * @param {{ snapshot?: unknown, from?: string, signal?: AbortSignal, timeoutMs?: number }} [options]
 */
export async function approveServiceRegistration(id, options = {}) {
  const sid = String(id ?? "");
  return dedupeRegistrationAction("approve", sid, () =>
    approveServiceRegistrationImpl(id, options)
  );
}

async function rejectServiceRegistrationImpl(id, options = {}) {
  const sid = String(id ?? "");
  const body = normalizeServiceRegistrationActionPayload(options);
  const snapRowRaw =
    typeof options.snapshot === "object" &&
    options.snapshot !== null &&
    !Array.isArray(options.snapshot)
      ? options.snapshot
      : shouldUseServiceRegistrationMocks()
        ? registrations.find((r) => r.id === sid)
        : undefined;

  if (shouldUseServiceRegistrationMocks()) {
    await delay(150);
    if (!snapRowRaw || typeof snapRowRaw !== "object") {
      throw new Error("Заявка не найдена.");
    }
    const snap = normalizeServiceRegistrationAdmin(snapRowRaw);
    assertServiceRegFromMatchesUi(snap, options.from);
    assertServiceRegistrationQueueAction(snap.status);
    registrations = registrations.map((r) =>
      r.id === sid
        ? mergeServiceRegistrationAdmin(r, {
            status: "rejected",
            rejection_reason: body.reason,
          })
        : r
    );
    return registrations.find((r) => r.id === sid);
  }

  if (snapRowRaw && typeof snapRowRaw === "object") {
    const snap = normalizeServiceRegistrationAdmin(snapRowRaw);
    assertServiceRegFromMatchesUi(snap, options.from);
    assertServiceRegistrationQueueAction(snap.status);
  }

  const { signal: userSignal, timeoutMs = 25_000 } = options ?? {};
  const { signal: timeoutSig, clear } = abortAfter(timeoutMs);
  const combined = combineAbortSignals(userSignal, timeoutSig);

  try {
    /** @type {unknown} */
    const raw = await apiClient.patch(
      `/v1/admin/car-service-applications/${encodeURIComponent(sid)}/reject`,
      { rejection_reason: body.reason || "Отклонено администратором" },
      { signal: combined, auth: true }
    );
    return mergeServiceRegistrationAdmin(
      snapRowRaw && typeof snapRowRaw === "object"
        ? snapRowRaw
        : { id: sid },
      raw ??
        {
          id: sid,
          status: "rejected",
          rejection_reason: body.reason,
        }
    );
  } catch (e) {
    rethrowAbortOrWrapAdminError(e);
  } finally {
    clear();
  }
}

/**
 * PATCH /v1/admin/car-service-applications/:id/reject
 * @param {string} id
 * @param {{ reason?: string, snapshot?: unknown, from?: string, signal?: AbortSignal, timeoutMs?: number }} [options]
 */
export async function rejectServiceRegistration(id, options = {}) {
  const sid = String(id ?? "");
  return dedupeRegistrationAction("reject", sid, () =>
    rejectServiceRegistrationImpl(id, options)
  );
}
