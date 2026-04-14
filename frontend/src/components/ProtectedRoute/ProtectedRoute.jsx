import { Navigate } from "react-router-dom";
import { useEffect, useState } from "react";
import { clearSession, ensureSession, hasAccessToken, isDevAuthBypassEnabled } from "../../services/authService";

function ProtectedRoute({ children }) {
  if (isDevAuthBypassEnabled()) {
    return children;
  }

  const token = hasAccessToken();
  const [isChecking, setIsChecking] = useState(true);
  const [isAllowed, setIsAllowed] = useState(false);

  useEffect(() => {
    let active = true;

    const check = async () => {
      if (!token) {
        if (active) {
          setIsAllowed(false);
          setIsChecking(false);
        }
        return;
      }

      try {
        await ensureSession();
        if (active) {
          setIsAllowed(true);
        }
      } catch {
        clearSession();
        if (active) {
          setIsAllowed(false);
        }
      } finally {
        if (active) {
          setIsChecking(false);
        }
      }
    };

    check();

    return () => {
      active = false;
    };
  }, [token]);

  if (isChecking) {
    return <div style={{ padding: 24 }}>Проверка сессии...</div>;
  }

  if (!isAllowed) {
    return <Navigate to="/auth" replace />;
  }

  return children;
}

export default ProtectedRoute;