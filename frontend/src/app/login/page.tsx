"use client";

import { useEffect, useRef, useState, useSyncExternalStore } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { apiFetch, warmBackend, ApiError } from "@/lib/api";
import {
  AuthCard,
  FormField,
  FormError,
  SubmitButton,
} from "@/components/auth/AuthCard";

const REMEMBERED_EMAIL_KEY = "theresa:remembered-email";

// Same useSyncExternalStore pattern ThemeToggle already uses for reading
// localStorage: a lazy useState initializer would run (and throw) during
// server-side prerendering where window doesn't exist, and setting it from
// inside an effect would mean the server-rendered and first-client-render
// HTML disagree, which React flags as a hydration mismatch. This reads a
// stable "" on the server and picks up the real value once mounted, without
// either problem - the subscribe callback is a no-op since nothing else in
// this tab changes the stored email while the form is open.
function readRememberedEmail(): string {
  return window.localStorage.getItem(REMEMBERED_EMAIL_KEY) ?? "";
}
function getServerSnapshot(): string {
  return "";
}
function subscribeNoop() {
  return () => {};
}

export default function LoginPage() {
  const router = useRouter();
  const rememberedEmail = useSyncExternalStore(subscribeNoop, readRememberedEmail, getServerSnapshot);
  const [emailOverride, setEmailOverride] = useState<string | null>(null);
  const email = emailOverride ?? rememberedEmail;
  const [password, setPassword] = useState("");
  const [rememberMe, setRememberMe] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const passwordRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    warmBackend();
  }, []);

  useEffect(() => {
    // Email's already filled in from a prior "remember me" - send focus
    // straight to the password field so a returning user really does just
    // need to type one thing. Depends on rememberedEmail (not []): on first
    // hydration useSyncExternalStore reports the SSR-safe "" snapshot, and
    // only resolves to the real localStorage value on the very next render -
    // an empty-deps effect would've already fired and missed it.
    if (rememberedEmail) {
      passwordRef.current?.focus();
    }
  }, [rememberedEmail]);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await apiFetch("/api/auth/login", {
        method: "POST",
        body: JSON.stringify({ email, password }),
      });
      if (rememberMe) {
        window.localStorage.setItem(REMEMBERED_EMAIL_KEY, email);
      } else {
        window.localStorage.removeItem(REMEMBERED_EMAIL_KEY);
      }
      router.push("/dashboard");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "something went wrong");
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthCard
      title="Welcome back"
      subtitle="Log in with your email and password to pick up your lessons where you left off."
      footer={
        <>
          Don&apos;t have an account?{" "}
          <Link href="/signup" className="text-[var(--color-accent)]">
            Sign up
          </Link>
        </>
      }
    >
      <form onSubmit={onSubmit} className="space-y-4">
        <FormField
          label="Email"
          placeholder="Email address"
          type="email"
          required
          value={email}
          onChange={(e) => setEmailOverride(e.target.value)}
        />
        <FormField
          ref={passwordRef}
          label="Password"
          placeholder="Password"
          type="password"
          required
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
        <div className="flex items-center justify-between">
          <label className="flex items-center gap-2 text-sm text-[var(--color-text-secondary)]">
            <input
              type="checkbox"
              checked={rememberMe}
              onChange={(e) => setRememberMe(e.target.checked)}
              className="h-4 w-4 rounded border-[var(--color-border)] accent-[var(--color-accent)]"
            />
            Remember me
          </label>
          <Link href="/forgot-password" className="text-sm text-[var(--color-accent)]">
            Forgot password?
          </Link>
        </div>
        {error && <FormError message={error} />}
        <SubmitButton loading={loading}>Log in</SubmitButton>
      </form>
    </AuthCard>
  );
}
