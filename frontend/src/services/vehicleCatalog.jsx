import { apiClient } from "./apiClient";

function listFromResponse(raw) {
  if (Array.isArray(raw)) return raw;
  if (Array.isArray(raw?.items)) return raw.items;
  return [];
}

export async function getVehicleMakes(options = {}) {
  const raw = await apiClient.get("/v1/vehicle-catalog/makes", {
    signal: options.signal,
    auth: true,
  });
  return listFromResponse(raw);
}

export async function getVehicleModels(makeID, options = {}) {
  if (!makeID) return [];

  const raw = await apiClient.get(
    `/v1/vehicle-catalog/makes/${encodeURIComponent(String(makeID))}/models`,
    {
      signal: options.signal,
      auth: true,
    }
  );
  return listFromResponse(raw);
}

export async function getVehicleGenerations(modelID, options = {}) {
  if (!modelID) return [];

  const raw = await apiClient.get(
    `/v1/vehicle-catalog/models/${encodeURIComponent(String(modelID))}/generations`,
    {
      signal: options.signal,
      auth: true,
    }
  );
  return listFromResponse(raw);
}
