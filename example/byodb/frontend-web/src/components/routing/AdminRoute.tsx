import { Navigate, Outlet } from "react-router-dom";

import { useUser } from "../../context/UserContext";

export function AdminRoute() {
  const { checkedSignIn, isSignedIn, isSuperUser } = useUser();

  if (!checkedSignIn) {
    return <div className="loading-state">Checking your admin access...</div>;
  }

  if (!isSignedIn()) {
    return <Navigate to="/login" replace />;
  }

  if (!isSuperUser()) {
    return <Navigate to="/console/getting-started" replace />;
  }

  return <Outlet />;
}
