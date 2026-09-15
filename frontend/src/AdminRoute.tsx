import { Navigate, Outlet } from "react-router-dom";
import { useApp } from "./lib";
export function AdminRoute() {
  return !!useApp().boot.profile?.is_admin ? (
    <Outlet />
  ) : (
    <Navigate to="/calendar" replace />
  );
}
