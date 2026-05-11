/**
 * Russian labels for internal role codes. Use only in UI —
 * comparisons and routing must keep using "USER" | "SERVICE" | "ADMIN".
 */
export const ROLE_LABELS = Object.freeze({
  USER: "Пользователь",
  SERVICE: "Автосервис",
  ADMIN: "Администратор",
});

/** @param {string | null | undefined} role */
export function getRoleLabel(role) {
  if (!role) return "";
  return ROLE_LABELS[role] ?? role;
}
