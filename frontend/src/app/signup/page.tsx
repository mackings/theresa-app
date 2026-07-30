"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { apiFetch, warmBackend, ApiError } from "@/lib/api";
import {
  AuthCard,
  FormField,
  FormError,
  SubmitButton,
} from "@/components/auth/AuthCard";

export default function SignupPage() {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [submitted, setSubmitted] = useState(false);

  useEffect(() => {
    warmBackend();
  }, []);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await apiFetch("/api/auth/signup", {
        method: "POST",
        body: JSON.stringify({ name, email, password }),
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
      <AuthCard title="Check your inbox" subtitle={`We sent a verification link to ${email}`}>
        <p className="text-sm text-[var(--color-text-secondary)]">
          Click the link in that email to verify your account, then{" "}
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
      title="Create your account"
      subtitle="Ask a question, paste a problem, or upload your notes — Theresa teaches any topic on a live board, out loud."
      footer={
        <>
          Already have an account?{" "}
          <Link href="/login" className="text-[var(--color-accent)]">
            Log in
          </Link>
        </>
      }
    >
      <form onSubmit={onSubmit} className="space-y-4">
        <FormField
          label="Name"
          placeholder="Name"
          type="text"
          required
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
        <FormField
          label="Email"
          placeholder="Email address"
          type="email"
          required
          value={email}
          onChange={(e) => setEmail(e.target.value)}
        />
        <FormField
          label="Password"
          placeholder="Password"
          type="password"
          required
          minLength={8}
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
        {error && <FormError message={error} />}
        <SubmitButton loading={loading}>Sign up</SubmitButton>
      </form>
    </AuthCard>
  );
}
