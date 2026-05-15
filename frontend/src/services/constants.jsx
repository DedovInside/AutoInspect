/**
 * Static configuration shared across the UI.
 *
 * These are NOT mocks — they are domain enums that the UI needs even when the
 * backend is fully wired (brand options for the upload form, the predefined
 * list of services that an autoservice can offer, the three damage levels,
 * etc.). They live here so pages stop importing from `mockData.jsx`.
 *
 * If/when any of these move to a real endpoint (e.g. GET /v1/brands), only
 * this module changes — pages keep their imports intact.
 */

import {
  CAR_BRANDS,
  SERVICE_OPTIONS,
  DAMAGE_LEVELS,
} from "./mockData";

/**
 * Марки, для которых на backend заведены отдельные ML-модели.
 * Временная заглушка; позже заменить на ответ API (например GET /v1/ml/supported-brands).
 */
export const MOCK_SUPPORTED_ML_BRANDS = Object.freeze([
  "Kia",
  "Hyundai",
  "Toyota",
  "Volkswagen",
]);

export { CAR_BRANDS, SERVICE_OPTIONS, DAMAGE_LEVELS };
