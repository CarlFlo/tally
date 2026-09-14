import { Navigate, Outlet } from "react-router-dom";
import { useApp } from "./lib";
export function AdminRoute() {
  return useApp().boot.profile?.id === "user0" ? (
    <Outlet />
  ) : (
    <Navigate to="/calendar" replace />
  );
}
