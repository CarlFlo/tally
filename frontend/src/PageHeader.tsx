import type { ReactNode } from "react";

export function PageHeader({ title, eyebrow, description, actions, className }: {
  title: string;
  eyebrow?: string;
  description?: ReactNode;
  actions?: ReactNode;
  className?: string;
}) {
  return (
    <div className={`page-heading${className ? ` ${className}` : ""}`}>
      <div>
        {eyebrow && <span className="eyebrow">{eyebrow}</span>}
        <h1>{title}<span className="accent">.</span></h1>
        {description && <p>{description}</p>}
      </div>
      {actions}
    </div>
  );
}
