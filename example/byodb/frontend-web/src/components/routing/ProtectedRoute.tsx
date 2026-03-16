import { Navigate, Outlet, useLocation } from "react-router-dom";

import { useUser } from "../../context/UserContext";

export function ProtectedRoute() {
  const { checkedSignIn, isSignedIn } = useUser();
  const location = useLocation();

  if (!checkedSignIn) {
    return <div className="loading-state">Checking your session...</div>;
  }

  if (!isSignedIn()) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />;
  }

  return <Outlet />;
}
