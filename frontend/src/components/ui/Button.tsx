import { forwardRef } from "react";

export type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";

const variantClasses: Record<ButtonVariant, string> = {
  primary:
    "bg-[var(--color-accent)] text-[var(--color-accent-foreground)] shadow-[var(--shadow-xs)] hover:opacity-90 disabled:opacity-60",
  secondary:
    "border border-[var(--color-border)] bg-[var(--color-surface-raised)] text-[var(--color-text-primary)] shadow-[var(--shadow-xs)] hover:bg-[var(--color-surface-hover)] disabled:opacity-60",
  ghost:
    "text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-hover)] hover:text-[var(--color-text-primary)] disabled:opacity-60",
  danger:
    "text-[var(--color-danger)] hover:bg-[var(--color-surface-hover)] disabled:opacity-60",
};

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  icon?: React.ReactNode;
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { variant = "secondary", icon, children, className, ...props },
  ref
) {
  return (
    <button
      ref={ref}
      type={props.type ?? "button"}
      className={`flex items-center justify-center gap-2 rounded-[var(--radius-md)] px-3.5 py-2 text-sm font-medium transition-colors ${variantClasses[variant]} ${className ?? ""}`}
      {...props}
    >
      {icon}
      {children}
    </button>
  );
});

interface IconButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  "aria-label": string;
}

export const IconButton = forwardRef<HTMLButtonElement, IconButtonProps>(function IconButton(
  { variant = "ghost", children, className, ...props },
  ref
) {
  return (
    <button
      ref={ref}
      type={props.type ?? "button"}
      className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-[var(--radius-md)] transition-colors ${variantClasses[variant]} ${className ?? ""}`}
      {...props}
    >
      {children}
    </button>
  );
});
