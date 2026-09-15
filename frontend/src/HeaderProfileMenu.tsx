import { NavLink } from "react-router-dom";
import { ChevronDown, Settings, Server, UserRound } from "lucide-react";
import { Avatar, SignOutButton, useApp } from "./lib";
import { usePopover } from "./usePopover";

export function HeaderProfileMenu() {
  const { boot } = useApp();
  const { open, setOpen, root, trigger } = usePopover();
  const profile = boot.profile!;
  return (
    <div className="header-popover" ref={root}>
      <button
        ref={trigger}
        className="header-profile"
        aria-label="Open profile menu"
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
          aria-label="Profile menu"
        >
          <NavLink to="/profile">
            <UserRound size={17} />
            Profile
          </NavLink>
          {!!profile.is_admin && (
            <>
              <NavLink to="/system">
                <Server size={17} />
                System
              </NavLink>
              <NavLink to="/settings">
                <Settings size={17} />
                Settings
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
