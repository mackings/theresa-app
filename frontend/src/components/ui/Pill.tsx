export function Pill({
  icon,
  children,
  className,
}: {
  icon?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-[var(--radius-full)] border border-[var(--color-border)] bg-[var(--color-surface-raised)] px-3 py-1 text-xs font-medium text-[var(--color-text-secondary)] shadow-[var(--shadow-xs)] ${className ?? ""}`}
    >
      {icon}
      {children}
    </span>
  );
}
