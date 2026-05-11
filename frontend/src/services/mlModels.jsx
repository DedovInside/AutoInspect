/**
 * Domain: ML models management (ADMIN).
 */

import { apiClient } from "./apiClient";
import { adminDevLog } from "./adminDebug";
import {
  buildMLModelUploadFormData,
  mergeMLModel,
  normalizeMLModel,
  normalizeMLModelList,
  normalizeMLUploadPayload,
  rethrowAbortOrWrapAdminError,
} from "./adminNormalize";
import { MOCK_ML_MODELS } from "./mockData";

export {
  buildMLModelUploadFormData,
  mergeMLModel,
  normalizeMLModel,
  normalizeMLModelList,
  normalizeMLUploadPayload,
} from "./adminNormalize";

function delay(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function shouldUseMockAdminMLModels() {
  if (import.meta.env.VITE_USE_MOCK_ADMIN_ML_MODELS === "true") return true;
  if (import.meta.env.VITE_USE_MOCK_ADMIN_ML_MODELS === "false") return false;
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

/** @type {ReturnType<typeof normalizeMLModel>[]} */
let models = MOCK_ML_MODELS.map((m) => normalizeMLModel(m));

/** @type {Map<string, Promise<unknown>>} */
const mutationInflight = new Map();

/**
 * @param {string} key
 * @param {() => Promise<unknown>} fn
 */
function dedupeMutation(key, fn) {
  if (mutationInflight.has(key)) {
    adminDevLog("ml.mutation.reuse_inflight", { key });
    return mutationInflight.get(key);
  }
  const p = Promise.resolve(fn()).finally(() => {
    mutationInflight.delete(key);
  });
  mutationInflight.set(key, p);
  return p;
}

/**
 * GET /v1/admin/ml-models
 * @param {{ signal?: AbortSignal, timeoutMs?: number }} [options]
 */
export async function listMLModels(options = {}) {
  const { signal: userSignal, timeoutMs = 25_000 } = options ?? {};

  if (shouldUseMockAdminMLModels()) {
    adminDevLog("ml.list.mock", {});
    await delay(150);
    return models.map((m) => normalizeMLModel(m));
  }

  const { signal: timeoutSig, clear } = abortAfter(timeoutMs);
  const combined = combineAbortSignals(userSignal, timeoutSig);

  try {
    /** @type {unknown} */
    const raw = await apiClient.get("/v1/admin/ml-models", {
      signal: combined,
      auth: true,
    });
    return normalizeMLModelList(raw);
  } catch (e) {
    rethrowAbortOrWrapAdminError(e);
  } finally {
    clear();
  }
}

/**
 * POST /v1/admin/ml-models (multipart)
 * @param {unknown} payload
 * @param {{ signal?: AbortSignal, timeoutMs?: number }} [options]
 */
export async function uploadMLModel(payload, options = {}) {
  return dedupeMutation("ml:upload", async () => {
    const clean = normalizeMLUploadPayload(payload);
    adminDevLog("ml.upload.start", { mock: shouldUseMockAdminMLModels() });

    if (shouldUseMockAdminMLModels()) {
      await delay(800);
      const created = normalizeMLModel({
        id: `m-${Date.now()}`,
        brand: clean.brand,
        model: clean.model,
        generation: clean.generation,
        years: clean.years,
        version: "",
        file: clean.modelFile.name || "model.pt",
        parts_catalog: clean.partsCatalogFile.name || "parts_catalog.json",
        accuracy: null,
        created_at: new Date().toISOString(),
        status: "active",
      });
      models = [created, ...models];
      adminDevLog("ml.upload.success", { mock: true });
      return created;
    }

    const { signal: userSignal, timeoutMs = 600_000 } = options ?? {};
    const { signal: timeoutSig, clear } = abortAfter(timeoutMs);
    const combined = combineAbortSignals(userSignal, timeoutSig);

    const form = buildMLModelUploadFormData(payload);

    try {
      /** @type {unknown} */
      const raw = await apiClient.post("/v1/admin/ml-models", form, {
        signal: combined,
        auth: true,
      });
      const basis = normalizeMLModel({
        brand: clean.brand,
        model: clean.model,
        generation: clean.generation,
        years: clean.years,
        status: "active",
      });
      const merged = mergeMLModel(basis, raw ?? {});
      adminDevLog("ml.upload.success", { mock: false });
      return merged;
    } catch (e) {
      rethrowAbortOrWrapAdminError(e);
    } finally {
      clear();
    }
  });
}

/**
 * POST /v1/admin/ml-models/:id/activate
 * @param {string} id
 * @param {{ signal?: AbortSignal, timeoutMs?: number, snapshot?: unknown }} [options]
 */
export async function activateMLModel(id, options = {}) {
  const sid = String(id ?? "");
  const snapshot = models.find((m) => m.id === sid) ?? options.snapshot;

  return dedupeMutation(`ml:activate:${sid}`, async () => {
    if (shouldUseMockAdminMLModels()) {
      await delay(120);
      models = models.map((m) =>
        m.id === sid ? normalizeMLModel({ ...m, status: "active" }) : m
      );
      return models.find((m) => m.id === sid);
    }

    const { signal: userSignal, timeoutMs = 25_000 } = options ?? {};
    const { signal: timeoutSig, clear } = abortAfter(timeoutMs);
    const combined = combineAbortSignals(userSignal, timeoutSig);

    try {
      /** @type {unknown} */
      const raw = await apiClient.post(
        `/v1/admin/ml-models/${encodeURIComponent(sid)}/activate`,
        {},
        { signal: combined, auth: true }
      );
      return mergeMLModel(snapshot ?? normalizeMLModel({ id: sid }), raw ?? {});
    } catch (e) {
      rethrowAbortOrWrapAdminError(e);
    } finally {
      clear();
    }
  });
}

/**
 * POST /v1/admin/ml-models/:id/deactivate
 * @param {string} id
 * @param {{ signal?: AbortSignal, timeoutMs?: number, snapshot?: unknown }} [options]
 */
export async function deactivateMLModel(id, options = {}) {
  const sid = String(id ?? "");
  const snapshot = models.find((m) => m.id === sid) ?? options.snapshot;

  return dedupeMutation(`ml:deactivate:${sid}`, async () => {
    if (shouldUseMockAdminMLModels()) {
      await delay(120);
      models = models.map((m) =>
        m.id === sid
          ? normalizeMLModel({ ...m, status: "deprecated" })
          : m
      );
      return models.find((m) => m.id === sid);
    }

    const { signal: userSignal, timeoutMs = 25_000 } = options ?? {};
    const { signal: timeoutSig, clear } = abortAfter(timeoutMs);
    const combined = combineAbortSignals(userSignal, timeoutSig);

    try {
      /** @type {unknown} */
      const raw = await apiClient.post(
        `/v1/admin/ml-models/${encodeURIComponent(sid)}/deactivate`,
        {},
        { signal: combined, auth: true }
      );
      return mergeMLModel(
        snapshot ?? normalizeMLModel({ id: sid }),
        raw ?? {}
      );
    } catch (e) {
      rethrowAbortOrWrapAdminError(e);
    } finally {
      clear();
    }
  });
}
