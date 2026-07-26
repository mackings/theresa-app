"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { CheckCircle2, Loader2, Sparkles } from "lucide-react";
import { AppShell } from "@/components/layout/AppShell";
import { Card } from "@/components/ui/Card";
import { Pill } from "@/components/ui/Pill";
import { getCreditBalance, formatNaira } from "@/lib/credits";

// Flutterwave redirects the browser here right after checkout, but the
// actual crediting happens via the webhook (server-to-server, independently
// verified) - which may land a moment before or after this page loads. So
// this polls briefly for the balance to change rather than assuming the
// redirect itself means the credit has landed yet.
const POLL_INTERVAL_MS = 1500;
const MAX_POLLS = 10;

export default function PaymentCallbackPage() {
  const [balanceKobo, setBalanceKobo] = useState<number | null>(null);
  const [confirmed, setConfirmed] = useState(false);

  useEffect(() => {
    let cancelled = false;
    let attempt = 0;
    const initialBalance = { current: null as number | null };

    async function poll() {
      try {
        const b = await getCreditBalance();
        if (cancelled) return;
        if (initialBalance.current === null) {
          initialBalance.current = b.balance_kobo;
        }
        setBalanceKobo(b.balance_kobo);
        if (b.balance_kobo > (initialBalance.current ?? 0) || attempt >= MAX_POLLS) {
          setConfirmed(true);
          return;
        }
      } catch {
        // keep trying
      }
      attempt++;
      if (attempt < MAX_POLLS && !cancelled) {
        setTimeout(poll, POLL_INTERVAL_MS);
      } else {
        setConfirmed(true);
      }
    }

    poll();
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <AppShell>
      <div className="mx-auto flex max-w-md flex-col items-center px-6 py-16">
        <Card className="w-full p-8 text-center">
          <div className="flex justify-center">
            <Pill icon={<Sparkles className="h-3.5 w-3.5 text-[var(--color-accent)]" />}>
              Voice credits
            </Pill>
          </div>

          <div
            className={`mx-auto mt-6 flex h-16 w-16 items-center justify-center rounded-full ${
              confirmed
                ? "bg-[var(--color-accent)]/10 text-[var(--color-accent)]"
                : "bg-[var(--color-surface-hover)] text-[var(--color-text-secondary)]"
            }`}
          >
            {confirmed ? (
              <CheckCircle2 className="h-8 w-8" />
            ) : (
              <Loader2 className="h-8 w-8 animate-spin" />
            )}
          </div>

          <h1 className="mt-5 text-xl font-bold text-[var(--color-text-primary)]">
            {confirmed ? "Balance updated" : "Confirming your payment…"}
          </h1>

          <p className="mt-2 text-3xl font-bold text-[var(--color-text-primary)]">
            {balanceKobo !== null ? formatNaira(balanceKobo) : "…"}
          </p>
          <p className="mt-1 text-sm text-[var(--color-text-secondary)]">current balance</p>

          {confirmed && (
            <Link
              href="/dashboard"
              className="mt-6 flex w-full items-center justify-center rounded-[var(--radius-full)] bg-[var(--color-accent)] px-4 py-3 text-sm font-semibold text-[var(--color-accent-foreground)] shadow-[var(--shadow-sm)] transition-opacity hover:opacity-90"
            >
              Back to dashboard
            </Link>
          )}
        </Card>
      </div>
    </AppShell>
  );
}
