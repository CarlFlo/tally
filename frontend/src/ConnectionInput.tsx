import { useState, type InputHTMLAttributes } from "react";
import { Eye, EyeOff } from "lucide-react";
import { useTranslation } from "react-i18next";

// These are service connection settings, not account sign-in fields. Keep the
// input type=text even when concealed to avoid password-manager autofill.
export function ConnectionInput({
  secret = false,
  hiddenByDefault = false,
  label,
  className,
  ...props
}: InputHTMLAttributes<HTMLInputElement> & {
  secret?: boolean;
  hiddenByDefault?: boolean;
  label: string;
}) {
  const { t } = useTranslation();
  const [hidden, setHidden] = useState(secret || hiddenByDefault);
  return (
    <div className="connection-input">
      <input
        {...props}
        type={props.type === "url" ? "url" : "text"}
        autoComplete="off"
        autoCorrect="off"
        autoCapitalize="none"
        spellCheck={false}
        data-1p-ignore="true"
        data-lpignore="true"
        data-bwignore="true"
        className={[className, hidden && "concealed-secret"]
          .filter(Boolean)
          .join(" ")}
      />
      {secret && (
        <button
          type="button"
          className="icon-button"
          aria-label={t(hidden ? "connection.show" : "connection.hide", { label })}
          aria-pressed={hidden}
          onClick={() => setHidden(!hidden)}
        >
          {hidden ? <Eye size={17} /> : <EyeOff size={17} />}
        </button>
      )}
    </div>
  );
}
