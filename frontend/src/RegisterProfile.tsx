import { useState, type FormEvent } from "react";
import { NavLink } from "react-router-dom";
import { api, Avatar, Busy, resetSession, useApp } from "./lib";

export function RegisterProfile() {
  const { boot, notify } = useApp();
  const [name, setName] = useState("");
  const [avatar, setAvatar] = useState("mint");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [busy, setBusy] = useState(false);
  async function submit(event: FormEvent) {
    event.preventDefault();
    if (password !== confirm) {
      notify("Passwords do not match", true);
      return;
    }
    setBusy(true);
    try {
      await api("/auth/register", "POST", { name, avatar, password });
      resetSession();
    } catch (e) {
      notify((e as Error).message, true);
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="picker-screen">
      <div className="picker-content">
        <span className="eyebrow">YOUR OWN LITTLE TV UNIVERSE</span>
        <h1>
          Add a profile<span className="accent">.</span>
        </h1>
        {boot.profiles.length >= boot.max_profiles ? (
          <p className="muted">All profile spaces are in use.</p>
        ) : boot.auth_mode === "oidc" ? (
          <a className="button primary" href="/auth/oidc/start">
            Continue with single sign-on
          </a>
        ) : (
          <form className="login-form" onSubmit={submit}>
            <label>
              Display name
              <input
                autoFocus
                required
                maxLength={80}
                autoComplete="nickname"
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </label>
            <div className="avatar-choices" aria-label="Choose avatar">
              {["mint", "violet", "amber", "rose", "blue"].map((color) => (
                <button
                  className={avatar === color ? "selected" : ""}
                  type="button"
                  key={color}
                  aria-label={`${color} avatar`}
                  aria-pressed={avatar === color}
                  onClick={() => setAvatar(color)}
                >
                  <Avatar
                    profile={{
                      id: "",
                      display_name: name || "You",
                      avatar: color,
                    }}
                  />
                </button>
              ))}
            </div>
            {boot.auth_mode === "local" && (
              <>
                <label>
                  New password
                  <input
                    type="password"
                    autoComplete="new-password"
                    required
                    minLength={boot.password_min}
                    maxLength={boot.password_max}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                  />
                </label>
                <label>
                  Confirm password
                  <input
                    type="password"
                    autoComplete="new-password"
                    required
                    maxLength={boot.password_max}
                    value={confirm}
                    onChange={(e) => setConfirm(e.target.value)}
                  />
                </label>
                <p className="small-text muted">
                  Use {boot.password_min}–{boot.password_max} characters.
                  Letters, numbers, and symbols are welcome.
                </p>
              </>
            )}
            <button className="button primary" disabled={busy}>
              {busy && <Busy />}Create profile
            </button>
          </form>
        )}
        <NavLink className="text-button" to="/login">
          Back to profiles
        </NavLink>
      </div>
    </div>
  );
}
