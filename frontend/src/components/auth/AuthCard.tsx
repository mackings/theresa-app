"use client";

import { useState } from "react";
import Link from "next/link";
import { AlertCircle, Eye, EyeOff } from "lucide-react";
import { ThemeToggle } from "@/components/theme/ThemeToggle";
import { SpeakingOrb } from "@/components/voice/SpeakingOrb";

export function AuthCard({
  title,
  subtitle,
  children,
  footer,
}: {
  title: string;
  subtitle?: string;
  children: React.ReactNode;
  footer?: React.ReactNode;
}) {
  return (
    <div className="flex min-h-full items-center justify-center bg-[var(--color-bg)] sm:px-6 sm:py-10">
      <div className="grid min-h-screen w-full max-w-6xl grid-cols-1 overflow-hidden bg-[var(--color-surface-raised)] sm:min-h-0 sm:rounded-[var(--radius-lg)] sm:border sm:border-[var(--color-border-subtle)] sm:shadow-[var(--shadow-lg)] lg:grid-cols-2">
        <div className="flex flex-col justify-between p-6 sm:p-12">
          <div className="flex items-center justify-between">
            <Link href="/" className="flex items-center gap-2">
              <div className="flex h-8 w-8 items-center justify-center rounded-[var(--radius-md)] bg-[var(--color-accent)] text-base font-bold text-[var(--color-accent-foreground)]">
                T
              </div>
              <span className="text-sm font-semibold text-[var(--color-text-primary)]">
                Theresa
              </span>
            </Link>
            <ThemeToggle />
          </div>

          <div className="my-10">
            <h1 className="text-3xl font-bold tracking-tight text-[var(--color-text-primary)] sm:text-4xl">
              {title}
            </h1>
            {subtitle && (
              <p className="mt-2 max-w-sm text-sm text-[var(--color-text-secondary)]">
                {subtitle}
              </p>
            )}

            <div className="mt-8">{children}</div>

            {footer && (
              <p className="mt-6 text-sm text-[var(--color-text-secondary)]">{footer}</p>
            )}
          </div>

          <p className="text-xs text-[var(--color-text-secondary)]">
            Need help?{" "}
            <a href="mailto:support@theresa.app" className="text-[var(--color-accent)]">
              support@theresa.app
            </a>
            <br />
            &copy; {new Date().getFullYear()} Theresa. All rights reserved.
          </p>
        </div>

        <div
          className="relative hidden overflow-hidden lg:flex lg:items-center lg:justify-center"
          style={{ background: "#0f231d" }}
        >
          <SpeakingOrb state="speaking" size={220} />

          <div className="absolute inset-x-8 bottom-8 rounded-[var(--radius-lg)] border border-white/10 bg-white/10 p-5 backdrop-blur-md">
            <p className="text-lg font-semibold text-white">
              Taught out loud, by Theresa.
            </p>
            <p className="mt-1 text-sm text-white/70">
              Ask a question, paste a problem, or upload your notes — Theresa
              teaches any topic on a live board, out loud.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}

export function FormField({
  label,
  placeholder,
  type,
  ...props
}: React.InputHTMLAttributes<HTMLInputElement> & { label: string }) {
  const [visible, setVisible] = useState(false);
  const isPassword = type === "password";

  return (
    <label className="block">
      <span className="sr-only">{label}</span>
      <div className="relative">
        <input
          placeholder={placeholder ?? label}
          {...props}
          type={isPassword ? (visible ? "text" : "password") : type}
          className={`w-full rounded-[var(--radius-lg)] bg-[var(--color-surface)] px-4 py-3 ${isPassword ? "pr-11" : ""} text-sm text-[var(--color-text-primary)] outline-none transition-shadow placeholder:text-[var(--color-text-secondary)] focus:shadow-[0_0_0_3px_color-mix(in_srgb,var(--color-accent)_20%,transparent)]`}
        />
        {isPassword && (
          <button
            type="button"
            onClick={() => setVisible((v) => !v)}
            aria-label={visible ? "Hide password" : "Show password"}
            className="absolute right-3.5 top-1/2 -translate-y-1/2 text-[var(--color-text-secondary)] transition-colors hover:text-[var(--color-text-primary)]"
          >
            {visible ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
          </button>
        )}
      </div>
    </label>
  );
}

export function FormError({ message }: { message: string }) {
  return (
    <p className="flex items-start gap-2 rounded-[var(--radius-md)] bg-[var(--color-surface-hover)] px-3 py-2 text-sm text-[var(--color-danger)]">
      <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
      <span>{message}</span>
    </p>
  );
}

export function SubmitButton({
  children,
  loading,
}: {
  children: React.ReactNode;
  loading?: boolean;
}) {
  return (
    <button
      type="submit"
      disabled={loading}
      className="w-full rounded-[var(--radius-full)] bg-[var(--color-accent)] px-4 py-3 text-sm font-medium text-[var(--color-accent-foreground)] shadow-[var(--shadow-xs)] transition-opacity hover:opacity-90 disabled:opacity-60"
    >
      {loading ? "Please wait…" : children}
    </button>
  );
}
