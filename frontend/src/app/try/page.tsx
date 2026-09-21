"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { AlertCircle, Loader2 } from "lucide-react";
import { apiFetch, ApiError } from "@/lib/api";
import { createSession } from "@/lib/sessions";

// The public, no-signup preview - see backend's DemoHandler for what "demo
// account" actually means (a real, ephemeral, password-less user with a
// tiny free-trial allowance). This page's only job is minting that account
// and a voice session, then landing on the exact same /session/[id] route a
// real signed-up user reaches - full AppShell/sidebar and all, not a
// stripped-down widget. The ?demo=1 flag just tells VoiceControls to point
// its out-of-credits CTA at signup instead of a /credits page a demo
// account can't meaningfully use.
export default function TryPage() {
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);
  const startedRef = useRef(false);

  useEffect(() => {
    // React Strict Mode double-invokes effects in dev - without this guard,
    // that would mint two ephemeral demo accounts (and burn two of the
    // per-IP rate limit's three slots) on a single mount.
    if (startedRef.current) return;
    startedRef.current = true;

    apiFetch("/api/demo/start", { method: "POST" })
      .then(() => createSession("voice"))
      .then((session) => router.replace(`/session/${session.id}?demo=1`))
      .catch((err: unknown) => {
        setError(
          err instanceof ApiError
            ? err.message
            : "We couldn't start a preview session. Please try again."
        );
      });
  }, [router]);

  if (error) {
    return (
      <div className="flex h-full items-center justify-center bg-[var(--color-bg)] px-6">
        <div className="flex max-w-sm flex-col items-center gap-3 text-center">
          <AlertCircle className="h-6 w-6 text-[var(--color-danger)]" />
          <p className="text-sm text-[var(--color-text-secondary)]">{error}</p>
          <Link
            href="/signup"
            className="rounded-[var(--radius-md)] bg-[var(--color-accent)] px-4 py-2 text-sm font-semibold text-[var(--color-accent-foreground)] shadow-[var(--shadow-xs)] transition-opacity hover:opacity-90"
          >
            Sign up instead
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-full items-center justify-center bg-[var(--color-bg)]">
      <span className="inline-flex items-center gap-2 text-sm text-[var(--color-text-secondary)]">
        <Loader2 className="h-4 w-4 animate-spin" />
        Starting your preview…
      </span>
    </div>
  );
}
