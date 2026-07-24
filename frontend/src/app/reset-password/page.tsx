"use client";

import { Suspense, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { apiFetch, ApiError } from "@/lib/api";
import {
  AuthCard,
  FormField,
  FormError,
  SubmitButton,
} from "@/components/auth/AuthCard";

function ResetPasswordContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const token = searchParams.get("token");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [done, setDone] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await apiFetch("/api/auth/reset-password", {
        method: "POST",
        body: JSON.stringify({ token, password }),
      });
      setDone(true);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "something went wrong");
    } finally {
      setLoading(false);
    }
  }

  if (!token) {
    return (
      <AuthCard title="Invalid reset link">
        <p className="text-sm text-[var(--color-text-secondary)]">
          This link is missing its reset token.{" "}
          <Link href="/forgot-password" className="text-[var(--color-accent)]">
            Request a new one
          </Link>
          .
        </p>
      </AuthCard>
    );
  }

  if (done) {
    return (
      <AuthCard title="Password updated">
        <p className="text-sm text-[var(--color-text-secondary)]">
          Your password has been reset.{" "}
          <button
            type="button"
            onClick={() => router.push("/login")}
            className="text-[var(--color-accent)]"
          >
            Log in
          </button>
          .
        </p>
      </AuthCard>
    );
  }

  return (
    <AuthCard title="Choose a new password" subtitle="Must be at least 8 characters.">
      <form onSubmit={onSubmit} className="space-y-4">
        <FormField
          label="New password"
          placeholder="New password"
          type="password"
          required
          minLength={8}
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
        {error && <FormError message={error} />}
        <SubmitButton loading={loading}>Reset password</SubmitButton>
      </form>
    </AuthCard>
  );
}

export default function ResetPasswordPage() {
  return (
    <Suspense>
      <ResetPasswordContent />
    </Suspense>
  );
}
