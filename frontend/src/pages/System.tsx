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
      <PageHeader title={t("system.title")} eyebrow={t("system.eyebrow")} description={t("system.description")} />
      <nav className="settings-tabs system-tabs" aria-label={t("system.sections")}>
        <NavLink to="/system/jobs">
          <Activity size={17} />
          {t("nav.jobs")}
        </NavLink>
        <NavLink to="/system/statistics">
          <ChartNoAxesCombined size={17} />
          {t("nav.statistics")}
        </NavLink>
        <NavLink to="/system/logs">
          <ScrollText size={17} />
          {t("nav.logs")}
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
