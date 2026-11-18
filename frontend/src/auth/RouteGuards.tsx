import { Navigate, Outlet, useLocation } from "react-router";

import { useAuth } from "./useAuth";

interface RedirectState {
  from?: string;
}

export function RequireAuth() {
  const { isAuthenticated } = useAuth();
  const location = useLocation();
  if (!isAuthenticated) {
    const state: RedirectState = { from: location.pathname + location.search };
    return <Navigate to="/login" replace state={state} />;
  }
  return <Outlet />;
}

export function GuestOnly() {
  const { isAuthenticated } = useAuth();
  const location = useLocation();
  if (isAuthenticated) {
    const state = location.state as RedirectState | null;
    return <Navigate to={state?.from ?? "/chats"} replace />;
  }
  return <Outlet />;
}
