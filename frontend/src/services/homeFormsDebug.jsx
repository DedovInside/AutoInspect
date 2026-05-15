/** Dev-only logging for home / admin form flows — no raw phone, email, or addresses. */

export function homeFormsDevLog(area, stage, details) {
  if (!import.meta.env.DEV) return;
  const safe =
    details && typeof details === "object" ? { ...details } : details ?? {};
  console.debug(`[AutoInspect HomeForms:${area}] ${stage}`, safe);
}
