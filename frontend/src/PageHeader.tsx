export function PageHeader({ title, eyebrow, description }: {
  title: string;
  eyebrow: string;
  description: string;
}) {
  return (
    <div className="page-heading">
      <div>
        <span className="eyebrow">{eyebrow}</span>
        <h1>{title}<span className="accent">.</span></h1>
        <p>{description}</p>
      </div>
    </div>
  );
}
