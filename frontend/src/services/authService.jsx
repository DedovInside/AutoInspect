const ACCESS_TOKEN_KEY = "access_token";
const REFRESH_TOKEN_KEY = "refresh_token";
const USER_KEY = "auth_user";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "";

function buildURL(path) {
  return `${API_BASE_URL}${path}`;
}

export function isDevAuthBypassEnabled() {
  return import.meta.env.DEV && import.meta.env.VITE_DEV_AUTH_BYPASS === "true";
}

async function parseJSON(response) {
  const raw = await response.text();

  if (!raw) {
    return null;
  }

  try {
    return JSON.parse(raw);
  } catch {
    throw new Error("Некорректный JSON-ответ от сервера");
  }
}

async function request(path, options = {}) {
  const response = await fetch(buildURL(path), {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(options.headers || {}),
    },
  });

  const payload = await parseJSON(response);

  if (!response.ok) {
    const message = payload?.message || `HTTP ${response.status}`;
    const error = new Error(message);
    error.status = response.status;
    error.code = payload?.code;
    throw error;
  }

  return payload;
}

function setStoredUser(user) {
  localStorage.setItem(USER_KEY, JSON.stringify(user));
}

function setStoredTokens(tokens) {
  localStorage.setItem(ACCESS_TOKEN_KEY, tokens.access_token);
  localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refresh_token);
}

function applyAuthResult(authResponse) {
  setStoredTokens(authResponse.tokens);
  setStoredUser(authResponse.user);
}

export function getAccessToken() {
  return localStorage.getItem(ACCESS_TOKEN_KEY);
}

export function getRefreshToken() {
  return localStorage.getItem(REFRESH_TOKEN_KEY);
}

export function getStoredUser() {
  const raw = localStorage.getItem(USER_KEY);
  if (!raw) {
    return null;
  }

  try {
    return JSON.parse(raw);
  } catch {
    return null;
  }
}

export function clearSession() {
  localStorage.removeItem(ACCESS_TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
  localStorage.removeItem(USER_KEY);
}

export function hasAccessToken() {
  return isDevAuthBypassEnabled() || Boolean(getAccessToken());
}

export async function startYandexOAuth() {
  const payload = await request("/v1/auth/yandex/start", {
    method: "GET",
  });

  return payload.auth_url;
}

export async function exchangeYandexCode(code, state) {
  const authResponse = await request("/v1/auth/oauth/yandex", {
    method: "POST",
    body: JSON.stringify({ code, state }),
  });

  applyAuthResult(authResponse);
  return authResponse;
}

export async function refreshSession() {
  const refreshToken = getRefreshToken();
  if (!refreshToken) {
    throw new Error("Нет refresh token");
  }

  const authResponse = await request("/v1/auth/refresh", {
    method: "POST",
    body: JSON.stringify({ refresh_token: refreshToken }),
  });

  applyAuthResult(authResponse);
  return authResponse;
}

export async function getMe() {
  const accessToken = getAccessToken();
  if (!accessToken) {
    throw new Error("Нет access token");
  }

  const user = await request("/v1/auth/me", {
    method: "GET",
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  });

  setStoredUser(user);
  return user;
}

export async function ensureSession() {
  if (!getAccessToken()) {
    return null;
  }

  try {
    return await getMe();
  } catch (error) {
    if (error?.status !== 401) {
      throw error;
    }
  }

  await refreshSession();
  return getMe();
}

export async function logout() {
  const accessToken = getAccessToken();
  const refreshToken = getRefreshToken();

  if (accessToken && refreshToken) {
    try {
      await request("/v1/auth/logout", {
        method: "POST",
        headers: {
          Authorization: `Bearer ${accessToken}`,
        },
        body: JSON.stringify({ refresh_token: refreshToken }),
      });
    } catch {
      // Сессию локально очищаем в любом случае.
    }
  }

  clearSession();
}
