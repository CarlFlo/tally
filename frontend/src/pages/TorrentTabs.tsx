import { Bot, History, ListOrdered, Search } from "lucide-react";
import { NavLink } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useApp } from "../lib";
import "../torrent-automation.css";

export function TorrentTabs() {
  const { t } = useTranslation();
  const { boot } = useApp();
  return (
    <nav className="settings-tabs torrent-tabs" aria-label={t("torrentAutomation.sections", { defaultValue: "Torrent search sections" })}>
      <NavLink to="/search" end>
        <Search size={17} />
        {t("torrentAutomation.searchTab", { defaultValue: "Search" })}
      </NavLink>
      {!!boot.profile?.is_admin && (
        <NavLink to="/search/automation">
          <Bot size={17} />
          {t("torrentAutomation.automationTab", { defaultValue: "Automation" })}
        </NavLink>
      )}
      <NavLink to="/search/runs">
        <History size={17} />
        {t("torrentAutomation.previousRunsTab", { defaultValue: "Previous Runs" })}
      </NavLink>
      {!!boot.profile?.is_admin && <NavLink to="/search/flows"><ListOrdered size={17} />{t("advancedFlows.title")}</NavLink>}
    </nav>
  );
}
