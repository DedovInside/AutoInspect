/** Dev-only analysis pipeline tracing (no file contents / small payloads). */

export function analysisDevLog(stage, details) {
  if (!import.meta.env.DEV) return;
  const safe =
    details && typeof details === "object" ? { ...details } : details ?? {};
  console.debug(`[AutoInspect Analysis] ${stage}`, safe);
}
