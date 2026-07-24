"use client";

import { Suspense, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import Link from "next/link";
import { apiFetch, ApiError } from "@/lib/api";
import { AuthCard } from "@/components/auth/AuthCard";

type Status = "verifying" | "success" | "error";

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
    <AuthCard title="Verification failed">
      <p className="text-sm text-[var(--color-text-secondary)]">{message}</p>
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
