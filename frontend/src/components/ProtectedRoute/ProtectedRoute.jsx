import { Navigate } from "react-router-dom";
import { useAuth } from "../../auth/AuthContext";
import Loader from "../Loader/Loader";

function ProtectedRoute({ children }) {
  const { isAuthenticated, isHydrating } = useAuth();

  if (isHydrating) {
    return (
      <div
        style={{
          minHeight: "60vh",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
        }}
      >
        <Loader label="Проверка сессии..." size="lg" />
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/auth" replace />;
  }

  return children;
}

export default ProtectedRoute;
