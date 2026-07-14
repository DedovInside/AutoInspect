import { apiClient } from "./apiClient";

function asObject(value) {
  return value && typeof value === "object" && !Array.isArray(value) ? value : {};
}

function unwrapReview(raw) {
  const root = asObject(raw);
  return root.review || root.item || root.data || raw;
}

export function normalizeCarServiceReview(raw) {
  const review = asObject(unwrapReview(raw));
  return {
    id: String(review.id || ""),
    repair_request_id: String(review.repair_request_id || review.repairRequestId || ""),
    car_service_profile_id: String(review.car_service_profile_id || review.carServiceProfileId || ""),
    user_id: String(review.user_id || review.userId || ""),
    rating: Number(review.rating || 0),
    author_name: String(review.author_name || review.authorName || "").trim(),
    comment: String(review.comment || "").trim(),
    created_at: review.created_at || review.createdAt || "",
    updated_at: review.updated_at || review.updatedAt || "",
  };
}

export function normalizeCarServiceReviewList(raw) {
  const root = asObject(raw);
  const source = Array.isArray(raw)
    ? raw
    : Array.isArray(root.items)
      ? root.items
      : Array.isArray(root.reviews)
        ? root.reviews
        : Array.isArray(root.data)
          ? root.data
          : [];
  return source.map(normalizeCarServiceReview).filter((review) => review.id);
}

export function reviewByRepairRequestId(reviews) {
  return Object.fromEntries(
    normalizeCarServiceReviewList(reviews)
      .filter((review) => review.repair_request_id)
      .map((review) => [review.repair_request_id, review])
  );
}

export async function createRepairRequestReview(repairRequestID, payload, options = {}) {
  const raw = await apiClient.post(
    `/v1/repair-requests/${encodeURIComponent(String(repairRequestID))}/review`,
    {
      rating: Number(payload.rating),
      author_name: String(payload.author_name || "").trim(),
      comment: String(payload.comment || "").trim(),
    },
    { ...options, auth: true }
  );
  return normalizeCarServiceReview(raw);
}

export async function updateRepairRequestReview(repairRequestID, payload, options = {}) {
  const raw = await apiClient.patch(
    `/v1/repair-requests/${encodeURIComponent(String(repairRequestID))}/review`,
    {
      rating: Number(payload.rating),
      author_name: String(payload.author_name || "").trim(),
      comment: String(payload.comment || "").trim(),
    },
    { ...options, auth: true }
  );
  return normalizeCarServiceReview(raw);
}

export async function deleteRepairRequestReview(repairRequestID, options = {}) {
  await apiClient.delete(
    `/v1/repair-requests/${encodeURIComponent(String(repairRequestID))}/review`,
    { ...options, auth: true }
  );
}

export async function getRepairRequestReview(repairRequestID, options = {}) {
  const raw = await apiClient.get(
    `/v1/repair-requests/${encodeURIComponent(String(repairRequestID))}/review`,
    { ...options, auth: true }
  );
  return normalizeCarServiceReview(raw);
}

export async function listMyCarServiceReviews(options = {}) {
  const raw = await apiClient.get("/v1/reviews/mine", {
    signal: options.signal,
    auth: true,
    query: {
      limit: options.limit ?? 100,
      offset: options.offset ?? 0,
    },
  });
  return normalizeCarServiceReviewList(raw);
}

export async function listCarServiceReviews(profileID, options = {}) {
  const raw = await apiClient.get(
    `/v1/car-services/${encodeURIComponent(String(profileID))}/reviews`,
    {
      signal: options.signal,
      auth: true,
      query: {
        limit: options.limit ?? 20,
        offset: options.offset ?? 0,
      },
    }
  );
  return normalizeCarServiceReviewList(raw);
}
