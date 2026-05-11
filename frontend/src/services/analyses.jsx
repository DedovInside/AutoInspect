/**
 * Domain: car damage analyses.
 * Production path: apiClient + normalization. Mock path: isolated (dev default).
 */

import { apiClient } from "./apiClient";
import { analysisDevLog } from "./analysisDebug";
import {
  normalizeAnalysisList,
  normalizeAnalysisResponse,
  normalizeUploadResponse,
} from "./analysisNormalize";
import { MOCK_ANALYSES, getMockAnalysis } from "./mockData";

export {
  isFailedTerminalAnalysisStatus,
  isSuccessTerminalAnalysisStatus,
  isTerminalAnalysisStatus,
  normalizeAnalysisList,
  normalizeAnalysisListItem,
  normalizeAnalysisResponse,
  normalizeAnalysisStatus,
  normalizeUploadResponse,
} from "./analysisNormalize";

function delay(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Dev: mocks on unless `VITE_USE_MOCK_ANALYSES=false`.
 * Prod: mocks only if `VITE_USE_MOCK_ANALYSES=true`.
 */
function shouldUseAnalysisMocks() {
  if (import.meta.env.VITE_USE_MOCK_ANALYSES === "true") return true;
  if (import.meta.env.VITE_USE_MOCK_ANALYSES === "false") return false;
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

/**
 * POST /v1/analyses (multipart) — start a new analysis.
 * @param {{ brand: string, model: string, generation: string, year: string, files: File[] }} payload
 * @param {{ signal?: AbortSignal, timeoutMs?: number }} [options]
 * @returns {Promise<{ analysis_id: string }>}
 */
export async function uploadImages(payload, options = {}) {
  const { brand, model, generation, year, files } = payload ?? {};
  const { signal: userSignal, timeoutMs = 180_000 } = options;

  analysisDevLog("upload.start", {
    fileCount: files?.length ?? 0,
    mock: shouldUseAnalysisMocks(),
  });

  if (shouldUseAnalysisMocks()) {
    await delay(1500);
    const out = normalizeUploadResponse({ analysis_id: "a-1042" });
    analysisDevLog("upload.success.mock", { analysis_id: out.analysis_id });
    return out;
  }

  const { signal: timeoutSig, clear: clearTimeoutTimer } = abortAfter(timeoutMs);
  const combined = combineAbortSignals(userSignal, timeoutSig);

  try {
    const form = new FormData();
    (files ?? []).forEach((f) => form.append("images", f));
    form.append("brand", brand ?? "");
    form.append("model", model ?? "");
    form.append("generation", generation ?? "");
    form.append("year", year ?? "");

    const raw = await apiClient.post("/v1/analyses", form, {
      signal: combined,
      auth: true,
    });

    const out = normalizeUploadResponse(raw);
    analysisDevLog("upload.success", { analysis_id: out.analysis_id });
    return out;
  } catch (err) {
    analysisDevLog("upload.fail", {
      name: /** @type {{ name?: string }} */ (err)?.name,
      status: /** @type {{ status?: number }} */ (err)?.status,
      code: /** @type {{ code?: string }} */ (err)?.code,
    });
    throw err;
  } finally {
    clearTimeoutTimer();
  }
}

/**
 * GET /v1/analyses/:id — fetch (or poll) analysis status & result.
 * @param {string} id
 * @param {{ signal?: AbortSignal, timeoutMs?: number }} [options]
 */
export async function getAnalysis(id, options = {}) {
  const { signal: userSignal, timeoutMs = 90_000 } = options;

  analysisDevLog("poll.request", { id, mock: shouldUseAnalysisMocks() });

  if (shouldUseAnalysisMocks()) {
    await delay(400);
    const raw = getMockAnalysis(id);
    const normalized = normalizeAnalysisResponse(
      { ...raw, status: raw?.status ?? "done" },
      { routeId: id }
    );
    const merged = { ...normalized, status: "done" };
    analysisDevLog("poll.mock", { status: merged.status });
    return merged;
  }

  const { signal: timeoutSig, clear: clearT } = abortAfter(timeoutMs);
  const combined = combineAbortSignals(userSignal, timeoutSig);

  try {
    const raw = await apiClient.get(
      `/v1/analyses/${encodeURIComponent(String(id))}`,
      {
        signal: combined,
        auth: true,
      }
    );
    const normalized = normalizeAnalysisResponse(raw, { routeId: id });
    analysisDevLog("poll.response", { id, status: normalized.status });
    return normalized;
  } catch (err) {
    analysisDevLog("poll.error", {
      id,
      name: /** @type {{ name?: string }} */ (err)?.name,
      status: /** @type {{ status?: number }} */ (err)?.status,
    });
    throw err;
  } finally {
    clearT();
  }
}

/**
 * GET /v1/analyses?limit=&cursor= — list of the current user's analyses.
 * @param {{ limit?: number, cursor?: string }} [query]
 * @param {{ signal?: AbortSignal, timeoutMs?: number }} [options]
 */
export async function getAnalysisHistory(query = {}, options = {}) {
  const { signal: userSignal, timeoutMs = 60_000 } = options;

  if (shouldUseAnalysisMocks()) {
    await delay(400);
    return normalizeAnalysisList(MOCK_ANALYSES);
  }

  const { signal: timeoutSig, clear: clearT } = abortAfter(timeoutMs);
  const combined = combineAbortSignals(userSignal, timeoutSig);

  try {
    const raw = await apiClient.get("/v1/analyses", {
      query,
      signal: combined,
      auth: true,
    });
    return normalizeAnalysisList(raw);
  } catch (err) {
    analysisDevLog("history.error", {
      status: /** @type {{ status?: number }} */ (err)?.status,
      code: /** @type {{ code?: string }} */ (err)?.code,
    });
    return [];
  } finally {
    clearT();
  }
}
