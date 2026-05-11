/** Dev-only admin panel integration logs — no payloads, contacts, filenames. */

export function adminDevLog(stage, details) {
  if (!import.meta.env.DEV) return;
  const safe =
    details && typeof details === "object" ? { ...details } : details ?? {};
  console.debug(`[AutoInspect Admin] ${stage}`, safe);
}
