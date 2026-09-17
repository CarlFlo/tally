import { useTranslation } from "react-i18next";
import { Avatar } from "./lib";
import "./profile-create.css";

const avatarColors = ["mint", "violet", "amber", "rose", "blue", "peach"];
const htmlColor = /^#[0-9A-Fa-f]{6}$/;

export function isValidHexColor(value: string) {
  return htmlColor.test(value);
}

export function normalizeHexColor(value: string) {
  const next = value.trim();
  if (!next || next.startsWith("#")) return next;
  return `#${next}`;
}

function previewAvatar(avatar: string, customColor: string) {
  return isValidHexColor(customColor) ? customColor : avatar;
}

export function ProfileAvatarPreview({
  name,
  avatar,
  customColor,
  locale,
}: {
  name: string;
  avatar: string;
  customColor: string;
  locale: string;
}) {
  const { t } = useTranslation();
  return (
    <div className="profile-create-preview" aria-live="polite">
      <Avatar
        profile={{
          id: "",
          display_name: name || t("profile.you"),
          avatar: previewAvatar(avatar, customColor),
          locale,
        }}
        large
      />
    </div>
  );
}

export function ProfileAvatarChoices({
  name,
  avatar,
  customColor,
  locale,
  onAvatarChange,
  onCustomColorChange,
}: {
  name: string;
  avatar: string;
  customColor: string;
  locale: string;
  onAvatarChange: (value: string) => void;
  onCustomColorChange: (value: string) => void;
}) {
  const { t } = useTranslation();
  const customSelected = isValidHexColor(customColor);
  return (
    <div className="profile-create-avatar-fields">
      <div className="avatar-choices" aria-label={t("accessibility.chooseAvatar")}>
        {avatarColors.map((color) => (
          <button
            className={!customSelected && avatar === color ? "selected" : ""}
            type="button"
            key={color}
            aria-label={t("accessibility.avatar", { color })}
            aria-pressed={!customSelected && avatar === color}
            onClick={() => {
              onAvatarChange(color);
              onCustomColorChange("");
            }}
          >
            <Avatar
              profile={{
                id: "",
                display_name: name || t("profile.you"),
                avatar: color,
                locale,
              }}
            />
          </button>
        ))}
      </div>
      <label>
        {t("profile.customAvatarColor")}
        <input
          value={customColor}
          pattern="#[0-9A-Fa-f]{6}"
          maxLength={7}
          placeholder="#4F46E5"
          spellCheck={false}
          onChange={(event) => onCustomColorChange(normalizeHexColor(event.target.value))}
        />
        <small className="muted">{t("profile.customAvatarHelp")}</small>
      </label>
    </div>
  );
}
