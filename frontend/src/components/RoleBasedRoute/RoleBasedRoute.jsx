import { Navigate } from "react-router-dom";
import { useAuth } from "../../auth/AuthContext";

function RoleBasedRoute({ children, allowedRoles = [] }) {
  const { role } = useAuth();

  if (allowedRoles.length === 0) {
    return children;
  }

  if (!role || !allowedRoles.includes(role)) {
    return <Navigate to="/home" replace />;
  }

  return children;
}

export default RoleBasedRoute;
