import { NavLink } from "react-router-dom";
import { ChevronDown, Settings, Server, UserRound } from "lucide-react";
import { Avatar, SignOutButton, useApp } from "./lib";
import { usePopover } from "./usePopover";
import { useTranslation } from "react-i18next";

export function HeaderProfileMenu() {
  const { t } = useTranslation();
  const { boot } = useApp();
  const { open, setOpen, root, trigger } = usePopover();
  const profile = boot.profile!;
  return (
    <div className="header-popover" ref={root}>
      <button
        ref={trigger}
        className="header-profile"
        aria-label={t("profile.openMenu")}
        aria-expanded={open}
        aria-controls="profile-menu"
        onClick={() => setOpen(!open)}
      >
        <Avatar profile={profile} />
        <strong>{profile.display_name}</strong>
        <ChevronDown size={15} />
      </button>
      {open && (
        <nav
          className="header-dropdown profile-dropdown"
          id="profile-menu"
          aria-label={t("profile.menu")}
        >
          <NavLink to="/profile">
            <UserRound size={17} />
            {t("profile.menuLink")}
          </NavLink>
          {!!profile.is_admin && (
            <>
              <NavLink to="/system">
                <Server size={17} />
                {t("nav.system")}
              </NavLink>
              <NavLink to="/settings">
                <Settings size={17} />
                {t("nav.settings")}
              </NavLink>
            </>
          )}
          <div className="dropdown-separator" />
          <SignOutButton className="text-button" />
        </nav>
      )}
    </div>
  );
}
