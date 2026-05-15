/** Development-only repair request tracing — no emails/phones logged by default beyond structure. */

export function repairDevLog(stage, details) {
  if (!import.meta.env.DEV) return;
  const safe =
    details && typeof details === "object" ? { ...details } : details ?? {};
  console.debug(`[AutoInspect Repairs] ${stage}`, safe);
}
