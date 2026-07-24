"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { apiFetch, ApiError } from "@/lib/api";
import {
  AuthCard,
  FormField,
  FormError,
  SubmitButton,
} from "@/components/auth/AuthCard";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await apiFetch("/api/auth/login", {
        method: "POST",
        body: JSON.stringify({ email, password }),
      });
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
          onChange={(e) => setEmail(e.target.value)}
        />
        <FormField
          label="Password"
          placeholder="Password"
          type="password"
          required
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
        <Link
          href="/forgot-password"
          className="inline-block text-sm text-[var(--color-accent)]"
        >
          Forgot password?
        </Link>
        {error && <FormError message={error} />}
        <SubmitButton loading={loading}>Log in</SubmitButton>
      </form>
    </AuthCard>
  );
}
