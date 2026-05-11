/**
 * Domain: ML model training requests (USER home → ADMIN queue).
 */

import { apiClient } from "./apiClient";
import { normalizeApiError } from "./apiFoundation";
import { homeFormsDevLog } from "./homeFormsDebug";
import {
  assertTrainingTransitionAllowed,
  mergeTrainingRequestAdmin,
  normalizeTrainingRequestAdmin,
  normalizeTrainingRequestAdminStatus,
  normalizeTrainingRequestList as normalizeAdminTrainingRequestList,
  normalizeTrainingRequestStatusPayload,
  rethrowAbortOrWrapAdminError,
} from "./adminNormalize";
import {
  mergeTrainingRequestResponse,
  normalizeTrainingRequest,
  normalizeTrainingRequestList,
  normalizeTrainingRequestPayload,
  trainingRequestPayloadToApiBody,
} from "./homeFormsNormalize";
import { MOCK_TRAINING_REQUESTS } from "./mockData";

export {
  mergeTrainingRequestResponse,
  normalizeTrainingRequest,
  normalizeTrainingRequestList,
  normalizeTrainingRequestPayload,
  trainingRequestPayloadToApiBody,
} from "./homeFormsNormalize";

export {
  assertTrainingTransitionAllowed,
  mergeTrainingRequestAdmin,
  normalizeTrainingRequestAdmin,
  normalizeTrainingRequestAdminStatus,
  normalizeTrainingRequestStatusPayload,
  rethrowAbortOrWrapAdminError,
} from "./adminNormalize";

function delay(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function shouldUseTrainingRequestMocks() {
  if (import.meta.env.VITE_USE_MOCK_TRAINING_REQUESTS === "true") {
    return true;
  }
  if (import.meta.env.VITE_USE_MOCK_TRAINING_REQUESTS === "false") {
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

function throwSubmitTrainingError(err) {
  const n = normalizeApiError(err);
  homeFormsDevLog("training", "submit.error", {
    aborted: n.code === "aborted",
    status: n.status,
    code: n.code,
  });
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
    typeof n.message === "string" && n.message.trim()
      ? n.message
      : "Не удалось отправить заявку на обучение"
  );
}

/** @type {Awaited<ReturnType<typeof normalizeTrainingRequestAdmin>>[]} */
let trainingRequests = MOCK_TRAINING_REQUESTS.map((row) =>
  normalizeTrainingRequestAdmin(row)
);

/** @type {Map<string, { to: string, c: AbortController, p: Promise<unknown> }>} */
const trainingStatusInflight = new Map();

function assertTrainingFromMatchesUi(snapshot, fromOpt) {
  if (snapshot == null) return;
  if (fromOpt == null || String(fromOpt).trim() === "") return;
  const cur = normalizeTrainingRequestAdminStatus(
    /** @type {{ status?: string }} */ (snapshot).status
  );
  const from = normalizeTrainingRequestAdminStatus(fromOpt);
  if (cur !== from) {
    throw new Error("Заявка уже обновлена. Обновите список.");
  }
}

/**
 * One in-flight status mutation per entity; same target reuses promise; new target aborts previous.
 *
 * @param {string} sid
 * @param {string} statusRaw
 * @param {(signal: AbortSignal | undefined) => Promise<unknown>} run
 */
function runExclusiveTrainingStatusMutation(sid, statusRaw, run) {
  const { status: canonTo } = normalizeTrainingRequestStatusPayload(statusRaw);
  const prev = trainingStatusInflight.get(sid);
  if (prev && prev.to === canonTo) return prev.p;

  prev?.c.abort();

  const c = new AbortController();
  const p = run(c.signal).finally(() => {
    const cur = trainingStatusInflight.get(sid);
    if (cur?.c === c) trainingStatusInflight.delete(sid);
  });

  trainingStatusInflight.set(sid, { to: canonTo, c, p });
  return p;
}

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
 * POST /v1/training-requests
 * @param {unknown} payload
 * @param {{ signal?: AbortSignal, timeoutMs?: number }} [options]
 */
export async function submitTrainingRequest(payload, options = {}) {
  const preview = normalizeTrainingRequestPayload(payload);
  homeFormsDevLog("training", "submit.enqueue", {
    mock: shouldUseTrainingRequestMocks(),
    hasBrandLen: preview.brand.length > 0,
  });

  return enqueueSubmit(async () => {
    const cleaned = normalizeTrainingRequestPayload(payload);

    if (shouldUseTrainingRequestMocks()) {
      await delay(260);
      const created = mergeTrainingRequestResponse(cleaned, {
        id: `tr-${Date.now()}`,
        submitted_at: new Date().toISOString(),
        status: "pending",
        brand: cleaned.brand,
        model: cleaned.model,
        generation: cleaned.generation,
        year: cleaned.year,
        description: cleaned.description,
      });
      trainingRequests = [
        normalizeTrainingRequestAdmin(created),
        ...trainingRequests,
      ];
      homeFormsDevLog("training", "submit.success", { mock: true });
      return created;
    }

    const { signal: userSignal, timeoutMs = 35_000 } = options ?? {};
    const { signal: timeoutSig, clear } = abortAfter(timeoutMs);
    const combined = combineAbortSignals(userSignal, timeoutSig);

    try {
      const body = trainingRequestPayloadToApiBody(cleaned);
      /** @type {unknown} */
      const raw = await apiClient.post("/v1/training-requests", body, {
        signal: combined,
        auth: true,
      });

      if (
        raw === null ||
        (typeof raw === "object" &&
          !Array.isArray(raw) &&
          Object.keys(/** @type {object} */ (raw ?? {})).length === 0)
      ) {
        homeFormsDevLog("training", "submit.partial_body", {});
      }

      const merged = mergeTrainingRequestResponse(cleaned, raw ?? {});
      homeFormsDevLog("training", "submit.success", { mock: false });
      return merged;
    } catch (e) {
      throwSubmitTrainingError(e);
    } finally {
      clear();
    }
  });
}

/**
 * GET /v1/admin/training-requests?status=
 * @param {{ status?: string, signal?: AbortSignal, timeoutMs?: number }} [options]
 */
export async function listTrainingRequests(options = {}) {
  const { status, signal: userSignal, timeoutMs = 25_000 } = options ?? {};

  if (shouldUseTrainingRequestMocks()) {
    await delay(120);
    const want =
      status != null && String(status).trim() !== ""
        ? normalizeTrainingRequestAdminStatus(status)
        : null;
    const list = !want
      ? trainingRequests
      : trainingRequests.filter(
          (r) => normalizeTrainingRequestAdminStatus(r.status) === want
        );
    return list.map((r) => normalizeTrainingRequestAdmin(r));
  }

  const { signal: timeoutSig, clear } = abortAfter(timeoutMs);
  const combined = combineAbortSignals(userSignal, timeoutSig);

  try {
    /** @type {unknown} */
    const raw = await apiClient.get("/v1/admin/training-requests", {
      signal: combined,
      auth: true,
      query: { status: status ?? "" },
    });
    return normalizeAdminTrainingRequestList(raw);
  } catch (e) {
    rethrowAbortOrWrapAdminError(e);
  } finally {
    clear();
  }
}

async function updateTrainingRequestStatusImpl(id, status, options, actionSignal) {
  const sid = String(id ?? "");
  const { status: canon } = normalizeTrainingRequestStatusPayload(status);
  const snapRowRaw =
    typeof options.snapshot === "object" &&
    options.snapshot !== null &&
    !Array.isArray(options.snapshot)
      ? options.snapshot
      : shouldUseTrainingRequestMocks()
        ? trainingRequests.find((r) => r.id === sid)
        : undefined;

  if (shouldUseTrainingRequestMocks()) {
    await delay(120);
    if (!snapRowRaw || typeof snapRowRaw !== "object") {
      throw new Error("Заявка не найдена.");
    }
    const snap = normalizeTrainingRequestAdmin(snapRowRaw);
    assertTrainingFromMatchesUi(snap, options.from);
    assertTrainingTransitionAllowed(snap.status, canon);
    trainingRequests = trainingRequests.map((r) =>
      r.id === sid ? mergeTrainingRequestAdmin(r, { status: canon }) : r
    );
    return trainingRequests.find((r) => r.id === sid);
  }

  if (snapRowRaw && typeof snapRowRaw === "object") {
    const snap = normalizeTrainingRequestAdmin(snapRowRaw);
    assertTrainingFromMatchesUi(snap, options.from);
    assertTrainingTransitionAllowed(snap.status, canon);
  }

  const { signal: userSignal, timeoutMs = 25_000 } = options ?? {};
  const { signal: timeoutSig, clear } = abortAfter(timeoutMs);
  const combined = combineAbortSignals(
    actionSignal,
    userSignal,
    timeoutSig
  );

  try {
    /** @type {unknown} */
    const raw = await apiClient.patch(
      `/v1/admin/training-requests/${encodeURIComponent(sid)}`,
      { status: canon },
      { signal: combined, auth: true }
    );
    return mergeTrainingRequestAdmin(
      snapRowRaw && typeof snapRowRaw === "object"
        ? snapRowRaw
        : { id: sid },
      raw ?? { id: sid, status: canon }
    );
  } catch (e) {
    rethrowAbortOrWrapAdminError(e);
  } finally {
    clear();
  }
}

/**
 * PATCH /v1/admin/training-requests/:id
 * @param {string} id
 * @param {string} status
 * @param {{ snapshot?: unknown, from?: string, signal?: AbortSignal, timeoutMs?: number }} [options]
 */
export async function updateTrainingRequestStatus(id, status, options = {}) {
  const sid = String(id ?? "");
  return runExclusiveTrainingStatusMutation(sid, status, (actionSignal) =>
    updateTrainingRequestStatusImpl(id, status, options ?? {}, actionSignal)
  );
}
