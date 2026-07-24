function getInitials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return "?";
  if (parts.length === 1) return parts[0][0].toUpperCase();
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}

export function Avatar({ name, size = 28 }: { name: string; size?: number }) {
  return (
    <div
      className="flex shrink-0 items-center justify-center rounded-full bg-[var(--color-accent)] font-medium text-[var(--color-accent-foreground)]"
      style={{ width: size, height: size, fontSize: Math.round(size * 0.4) }}
    >
      {getInitials(name)}
    </div>
  );
}
