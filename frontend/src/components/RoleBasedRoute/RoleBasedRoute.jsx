import { Navigate } from "react-router-dom";
import { getUserRole } from "../../services/authService";

function RoleBasedRoute({ children, allowedRoles = [] }) {
  const role = getUserRole();

  if (allowedRoles.length === 0) {
    return children;
  }

  if (!allowedRoles.includes(role)) {
    return <Navigate to="/home" replace />;
  }

  return children;
}

export default RoleBasedRoute;