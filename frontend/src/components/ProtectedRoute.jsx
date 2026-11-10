import { Navigate, useLocation } from "react-router-dom";
import { useAuth } from "../context/AuthContext";

export default function ProtectedRoute({ children }) {
  const { isAuthenticated } = useAuth();
  const location = useLocation();
  if (!isAuthenticated) {
    // Keep the requested path so the admin lands there after signing in.
    return <Navigate to="/admin/login" replace state={{ from: location.pathname + location.search }} />;
  }
  return children;
}
