import { ArrowRight, Plus } from "lucide-react";
import { useState } from "react";
import { Navigate, NavLink, useNavigate, useParams } from "react-router-dom";
import { api, Avatar, resetSession, type Boot, type Profile } from "./lib";
import { LoginCredentials } from "./LoginCredentials";
import { Logo } from "./Logo";

export function ProfilePicker({
  boot,
  notify,
}: {
  boot: Boot;
  notify: (s: string, e?: boolean) => void;
}) {
  const { profileId } = useParams();
  const navigate = useNavigate();
  const selected =
    boot.auth_mode === "local"
      ? boot.profiles.find((profile) => profile.id === profileId)
      : undefined;
  const [busy, setBusy] = useState(false);
  if (profileId && !selected) return <Navigate to="/login" replace />;
  async function choose(profile: Profile) {
    if (boot.auth_mode === "local") {
      navigate("/login/" + profile.id);
      return;
    }
    if (boot.auth_mode === "oidc") {
      window.location.assign("/auth/oidc/start");
      return;
    }
    setBusy(true);
    try {
      await api("/profiles/select", "POST", { profile: profile.id });
      resetSession();
    } catch (e) {
      notify((e as Error).message, true);
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="picker-screen">
      <Logo />
      <div className="picker-content">
        <span className="eyebrow">MAKE YOURSELF AT HOME</span>
        <h1>
          {selected
            ? selected.has_password
              ? `Welcome back, ${selected.display_name}.`
              : `Set up your password, ${selected.display_name}.`
            : "Who's keeping up?"}
        </h1>
        <p className="muted">
          Your shows, your progress, your next great episode.
        </p>
        {selected ? (
          <LoginCredentials key={selected.id} profile={selected} />
        ) : boot.auth_mode === "oidc" ? (
          <a className="button primary" href="/auth/oidc/start">
            Continue with single sign-on
            <ArrowRight size={18} />
          </a>
        ) : (
          <div className="profile-grid">
            {boot.profiles.map((p) => (
              <button
                className="picker-profile"
                key={p.id}
                disabled={busy}
                onClick={() => choose(p)}
              >
                <Avatar profile={p} large />
                <strong>{p.display_name}</strong>
              </button>
            ))}
            <NavLink
              to="/login/new"
              className="picker-profile add-profile-tile"
            >
              <span className="add-profile-icon">
                <Plus size={36} />
              </span>
              <strong>Add profile</strong>
            </NavLink>
          </div>
        )}
      </div>
      <span className="picker-footer">
        A personal space for the shows you love.
      </span>
    </div>
  );
}
