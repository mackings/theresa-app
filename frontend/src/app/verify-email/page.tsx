"use client";

import { Suspense, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import Link from "next/link";
import { apiFetch, ApiError } from "@/lib/api";
import {
  AuthCard,
  FormField,
  FormError,
  SubmitButton,
} from "@/components/auth/AuthCard";

type Status = "verifying" | "success" | "error";

// The only way to get here failing is an expired or already-used link, most
// commonly - previously a genuine dead end (a static error message and
// nothing else), since the only way to actually get a fresh link was to
// stumble into it as a side effect of a failed login attempt. This is the
// discoverable path: right on the page a student is actually looking at
// when they need it.
function ResendVerificationForm() {
  const [email, setEmail] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [submitted, setSubmitted] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await apiFetch("/api/auth/resend-verification", {
        method: "POST",
        body: JSON.stringify({ email }),
      });
      setSubmitted(true);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "something went wrong");
    } finally {
      setLoading(false);
    }
  }

  if (submitted) {
    return (
      <p className="text-sm text-[var(--color-text-secondary)]">
        If {email} has an account that still needs verifying, we&apos;ve sent a new
        link - check your inbox (and your spam folder if you don&apos;t see it soon).
      </p>
    );
  }

  return (
    <form onSubmit={onSubmit} className="space-y-3">
      <p className="text-sm text-[var(--color-text-secondary)]">
        Enter your email and we&apos;ll send you a new verification link.
      </p>
      <FormField
        label="Email"
        placeholder="Email address"
        type="email"
        required
        value={email}
        onChange={(e) => setEmail(e.target.value)}
      />
      {error && <FormError message={error} />}
      <SubmitButton loading={loading}>Resend verification email</SubmitButton>
    </form>
  );
}

function VerifyEmailContent() {
  const searchParams = useSearchParams();
  const token = searchParams.get("token");
  const [status, setStatus] = useState<Status>("verifying");
  const [message, setMessage] = useState("");

  useEffect(() => {
    if (!token) return;

    apiFetch("/api/auth/verify-email", {
      method: "POST",
      body: JSON.stringify({ token }),
    })
      .then(() => setStatus("success"))
      .catch((err) => {
        setStatus("error");
        setMessage(err instanceof ApiError ? err.message : "something went wrong");
      });
  }, [token]);

  if (!token) {
    return (
      <AuthCard title="Verification failed">
        <p className="text-sm text-[var(--color-text-secondary)]">
          missing verification token
        </p>
        <div className="mt-6">
          <ResendVerificationForm />
        </div>
      </AuthCard>
    );
  }

  if (status === "verifying") {
    return (
      <AuthCard title="Verifying your email…">
        <p className="text-sm text-[var(--color-text-secondary)]">
          One moment.
        </p>
      </AuthCard>
    );
  }

  if (status === "success") {
    return (
      <AuthCard title="Email verified">
        <p className="text-sm text-[var(--color-text-secondary)]">
          Your account is ready.{" "}
          <Link href="/login" className="text-[var(--color-accent)]">
            Log in
          </Link>
          .
        </p>
      </AuthCard>
    );
  }

  return (
    <AuthCard
      title="Verification failed"
      footer={
        <>
          Already verified?{" "}
          <Link href="/login" className="text-[var(--color-accent)]">
            Log in
          </Link>
        </>
      }
    >
      <p className="text-sm text-[var(--color-text-secondary)]">{message}</p>
      <div className="mt-6">
        <ResendVerificationForm />
      </div>
    </AuthCard>
  );
}

export default function VerifyEmailPage() {
  return (
    <Suspense>
      <VerifyEmailContent />
    </Suspense>
  );
}
