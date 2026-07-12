import { apiClient } from "./apiClient";

function listFromResponse(raw) {
  if (Array.isArray(raw)) return raw;
  if (Array.isArray(raw?.items)) return raw.items;
  return [];
}

function normalizeCatalogItem(item) {
  if (!item || typeof item !== "object") return null;
  return {
    ...item,
    id: String(item.id ?? ""),
    name: String(item.name ?? ""),
    slug: String(item.slug ?? ""),
    is_active: Boolean(item.is_active),
  };
}

function normalizeCatalogList(raw) {
  return listFromResponse(raw)
    .map((item) => normalizeCatalogItem(item))
    .filter((item) => item && item.id);
}

function cleanNameBody(input) {
  return {
    name: String(input?.name ?? "").trim(),
  };
}

function cleanGenerationBody(input) {
  const body = {
    model_id: String(input?.model_id ?? input?.modelID ?? "").trim(),
    name: String(input?.name ?? "").trim(),
    year_from: Number(input?.year_from ?? input?.yearFrom),
  };

  const yearTo = input?.year_to ?? input?.yearTo;
  if (yearTo !== undefined && yearTo !== null && String(yearTo).trim() !== "") {
    body.year_to = Number(yearTo);
  }

  return body;
}

export async function listAdminVehicleMakes(options = {}) {
  const raw = await apiClient.get("/v1/admin/vehicle-catalog/makes", {
    signal: options.signal,
    auth: true,
  });
  return normalizeCatalogList(raw);
}

export async function createAdminVehicleMake(input, options = {}) {
  const raw = await apiClient.post(
    "/v1/admin/vehicle-catalog/makes",
    cleanNameBody(input),
    {
      signal: options.signal,
      auth: true,
    }
  );
  return normalizeCatalogItem(raw);
}

export async function updateAdminVehicleMake(id, input, options = {}) {
  const raw = await apiClient.patch(
    `/v1/admin/vehicle-catalog/makes/${encodeURIComponent(String(id))}`,
    cleanNameBody(input),
    {
      signal: options.signal,
      auth: true,
    }
  );
  return normalizeCatalogItem(raw);
}

export async function setAdminVehicleMakeActive(id, isActive, options = {}) {
  await apiClient.patch(
    `/v1/admin/vehicle-catalog/makes/${encodeURIComponent(String(id))}/active`,
    { is_active: Boolean(isActive) },
    {
      signal: options.signal,
      auth: true,
    }
  );
}

export async function listAdminVehicleModels(makeID, options = {}) {
  if (!makeID) return [];
  const raw = await apiClient.get(
    `/v1/admin/vehicle-catalog/makes/${encodeURIComponent(String(makeID))}/models`,
    {
      signal: options.signal,
      auth: true,
    }
  );
  return normalizeCatalogList(raw);
}

export async function createAdminVehicleModel(input, options = {}) {
  const raw = await apiClient.post(
    "/v1/admin/vehicle-catalog/models",
    {
      make_id: String(input?.make_id ?? input?.makeID ?? "").trim(),
      ...cleanNameBody(input),
    },
    {
      signal: options.signal,
      auth: true,
    }
  );
  return normalizeCatalogItem(raw);
}

export async function updateAdminVehicleModel(id, input, options = {}) {
  const raw = await apiClient.patch(
    `/v1/admin/vehicle-catalog/models/${encodeURIComponent(String(id))}`,
    {
      make_id: String(input?.make_id ?? input?.makeID ?? "").trim(),
      ...cleanNameBody(input),
    },
    {
      signal: options.signal,
      auth: true,
    }
  );
  return normalizeCatalogItem(raw);
}

export async function setAdminVehicleModelActive(id, isActive, options = {}) {
  await apiClient.patch(
    `/v1/admin/vehicle-catalog/models/${encodeURIComponent(String(id))}/active`,
    { is_active: Boolean(isActive) },
    {
      signal: options.signal,
      auth: true,
    }
  );
}

export async function listAdminVehicleGenerations(modelID, options = {}) {
  if (!modelID) return [];
  const raw = await apiClient.get(
    `/v1/admin/vehicle-catalog/models/${encodeURIComponent(String(modelID))}/generations`,
    {
      signal: options.signal,
      auth: true,
    }
  );
  return normalizeCatalogList(raw);
}

export async function createAdminVehicleGeneration(input, options = {}) {
  const raw = await apiClient.post(
    "/v1/admin/vehicle-catalog/generations",
    cleanGenerationBody(input),
    {
      signal: options.signal,
      auth: true,
    }
  );
  return normalizeCatalogItem(raw);
}

export async function updateAdminVehicleGeneration(id, input, options = {}) {
  const raw = await apiClient.patch(
    `/v1/admin/vehicle-catalog/generations/${encodeURIComponent(String(id))}`,
    cleanGenerationBody(input),
    {
      signal: options.signal,
      auth: true,
    }
  );
  return normalizeCatalogItem(raw);
}

export async function setAdminVehicleGenerationActive(id, isActive, options = {}) {
  await apiClient.patch(
    `/v1/admin/vehicle-catalog/generations/${encodeURIComponent(String(id))}/active`,
    { is_active: Boolean(isActive) },
    {
      signal: options.signal,
      auth: true,
    }
  );
}
