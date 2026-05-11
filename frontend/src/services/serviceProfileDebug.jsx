/** Development-only tracing for service profile (no passwords, phones, emails in logs). */

export function serviceProfileDevLog(stage, details) {
  if (!import.meta.env.DEV) return;
  const safe =
    details && typeof details === "object" ? { ...details } : details ?? {};
  console.debug(`[AutoInspect ServiceProfile] ${stage}`, safe);
}
