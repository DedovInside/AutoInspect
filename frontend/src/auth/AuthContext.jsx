import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import {
  ensureSession,
  getStoredUser,
  hasAccessToken,
  isDevAuthBypassEnabled,
  logout as logoutService,
} from "../services/authService";

const DEV_ROLE = import.meta.env.VITE_DEV_ROLE || "USER";

function buildDevUser() {
  return {
    id: "dev-user",
    email: "dev@autoinspect.local",
    username: "Локальный пользователь",
    avatar_url: "",
    role: DEV_ROLE,
  };
}

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const bypass = isDevAuthBypassEnabled();

  const [user, setUser] = useState(() =>
    bypass ? buildDevUser() : getStoredUser()
  );
  const [isHydrating, setIsHydrating] = useState(
    () => !bypass && hasAccessToken()
  );

  useEffect(() => {
    if (bypass) return undefined;
    if (!hasAccessToken()) {
      setIsHydrating(false);
      return undefined;
    }

    let cancelled = false;
    setIsHydrating(true);

    ensureSession()
      .then((nextUser) => {
        if (!cancelled) setUser(nextUser ?? null);
      })
      .finally(() => {
        if (!cancelled) setIsHydrating(false);
      });

    return () => {
      cancelled = true;
    };
  }, [bypass]);

  const syncFromStorage = useCallback(() => {
    if (bypass) {
      setUser(buildDevUser());
      return;
    }
    setUser(getStoredUser());
  }, [bypass]);

  const logout = useCallback(async () => {
    await logoutService();
    setUser(null);
  }, []);

  const value = useMemo(() => {
    const role = user?.role || (bypass ? DEV_ROLE : null);
    return {
      user,
      role,
      isAuthenticated: Boolean(user) || bypass,
      isHydrating,
      syncFromStorage,
      logout,
    };
  }, [user, isHydrating, bypass, syncFromStorage, logout]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used inside <AuthProvider>");
  }
  return ctx;
}
