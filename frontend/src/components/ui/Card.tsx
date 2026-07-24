export function Card({
  className,
  children,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={`rounded-[var(--radius-lg)] border border-[var(--color-border-subtle)] bg-[var(--color-surface-raised)] shadow-[var(--shadow-sm)] ${className ?? ""}`}
      {...props}
    >
      {children}
    </div>
  );
}

export function CardHeader({
  icon,
  title,
  action,
}: {
  icon?: React.ReactNode;
  title: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between gap-2 px-4 pt-4">
      <div className="flex items-center gap-2 text-xs font-medium uppercase tracking-wide text-[var(--color-text-secondary)]">
        {icon}
        <span>{title}</span>
      </div>
      {action}
    </div>
  );
}
