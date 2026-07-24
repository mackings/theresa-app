export function FeatureShowcaseCard({
  preview,
  title,
  description,
  className,
  ...props
}: {
  preview: React.ReactNode;
  title: string;
  description: string;
} & React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={`overflow-hidden rounded-[var(--radius-lg)] border border-[var(--color-border-subtle)] bg-[var(--color-surface-raised)] shadow-[var(--shadow-sm)] ${className ?? ""}`}
      {...props}
    >
      <div className="flex min-h-[140px] items-center justify-center bg-[var(--color-surface)] p-5">
        {preview}
      </div>
      <div className="p-5">
        <p className="text-base font-semibold text-[var(--color-text-primary)]">{title}</p>
        <p className="mt-1 text-sm text-[var(--color-text-secondary)]">{description}</p>
      </div>
    </div>
  );
}
