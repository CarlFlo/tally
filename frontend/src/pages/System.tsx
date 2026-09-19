import { NavLink, Navigate, Route, Routes } from "react-router-dom";
import { Activity, ChartNoAxesCombined, ScrollText } from "lucide-react";
import { JobsPage, StatisticsPage } from "./Operations";
import { LogsPage } from "./Logs";
import { PageHeader } from "../PageHeader";
import { useTranslation } from "react-i18next";

export function SystemPage() {
  const { t } = useTranslation();
  return (
    <div className="page system-page">
      <PageHeader title={t("system.title")} description={t("system.description")} />
      <nav className="settings-tabs system-tabs" aria-label={t("system.sections")}>
        <NavLink to="/admin/operations/jobs">
          <Activity size={17} />
          {t("nav.jobs")}
        </NavLink>
        <NavLink to="/admin/operations/statistics">
          <ChartNoAxesCombined size={17} />
          {t("nav.statistics")}
        </NavLink>
        <NavLink to="/admin/operations/logs">
          <ScrollText size={17} />
          {t("nav.logs")}
        </NavLink>
      </nav>
      <div className="system-content">
        <Routes>
          <Route path="jobs" element={<JobsPage />} />
          <Route path="statistics" element={<StatisticsPage />} />
          <Route path="logs" element={<LogsPage />} />
          <Route path="*" element={<Navigate to="/admin/operations/jobs" replace />} />
        </Routes>
      </div>
    </div>
  );
}
