import { NavLink, Navigate, Route, Routes } from "react-router-dom";
import { Activity, ChartNoAxesCombined, ScrollText } from "lucide-react";
import { JobsPage, StatisticsPage } from "./Operations";
import { LogsPage } from "./Logs";
import { PageHeader } from "../PageHeader";

export function SystemPage() {
  return (
    <div className="page system-page">
      <PageHeader title="System" eyebrow="BEHIND THE SCENES" description="Monitor jobs, statistics, and activity." />
      <nav className="settings-tabs system-tabs" aria-label="System sections">
        <NavLink to="/system/jobs">
          <Activity size={17} />
          Jobs
        </NavLink>
        <NavLink to="/system/statistics">
          <ChartNoAxesCombined size={17} />
          Statistics
        </NavLink>
        <NavLink to="/system/logs">
          <ScrollText size={17} />
          Logs
        </NavLink>
      </nav>
      <Routes>
        <Route path="jobs" element={<JobsPage />} />
        <Route path="statistics" element={<StatisticsPage />} />
        <Route path="logs" element={<LogsPage />} />
        <Route path="*" element={<Navigate to="/system/jobs" replace />} />
      </Routes>
    </div>
  );
}
