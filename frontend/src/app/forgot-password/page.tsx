"use client";

import { useState } from "react";
import Link from "next/link";
import { apiFetch, ApiError } from "@/lib/api";
import {
  AuthCard,
  FormField,
  FormError,
  SubmitButton,
} from "@/components/auth/AuthCard";

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [submitted, setSubmitted] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await apiFetch("/api/auth/forgot-password", {
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
      <AuthCard title="Check your inbox" subtitle={`If ${email} has an account, we've sent a password reset link.`}>
        <p className="text-sm text-[var(--color-text-secondary)]">
          Click the link in that email to choose a new password, then{" "}
          <Link href="/login" className="text-[var(--color-accent)]">
            log in
          </Link>
          .
        </p>
      </AuthCard>
    );
  }

  return (
    <AuthCard
      title="Reset your password"
      subtitle="Enter the email address on your account and we'll send you a link to reset your password."
      footer={
        <>
          Remembered it?{" "}
          <Link href="/login" className="text-[var(--color-accent)]">
            Log in
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
          onChange={(e) => setEmail(e.target.value)}
        />
        {error && <FormError message={error} />}
        <SubmitButton loading={loading}>Send reset link</SubmitButton>
      </form>
    </AuthCard>
  );
}
