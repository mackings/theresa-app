"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { AlertCircle, ArrowLeft, Check, ShieldCheck } from "lucide-react";
import { apiFetch } from "@/lib/api";
import { AppShell } from "@/components/layout/AppShell";
import { Card } from "@/components/ui/Card";
import { Pill } from "@/components/ui/Pill";
import { getCreditBalance, initiatePurchase, formatNaira } from "@/lib/credits";

const PRESET_AMOUNTS = [1000, 2500, 5000, 10000];
const MIN_AMOUNT = 1000;

export default function CreditsPage() {
  const router = useRouter();
  const [balanceKobo, setBalanceKobo] = useState<number | null>(null);
  const [freeTrialSecondsLeft, setFreeTrialSecondsLeft] = useState(0);
  const [selected, setSelected] = useState<number>(PRESET_AMOUNTS[0]);
  const [customAmount, setCustomAmount] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    apiFetch("/api/auth/me").catch(() => router.replace("/login"));
    getCreditBalance()
      .then((b) => {
        setBalanceKobo(b.balance_kobo);
        setFreeTrialSecondsLeft(b.free_trial_seconds_remaining);
      })
      .catch(() => {});
  }, [router]);

  const amount = customAmount ? parseInt(customAmount, 10) || 0 : selected;

  function handleCustomAmountChange(e: React.ChangeEvent<HTMLInputElement>) {
    setCustomAmount(e.target.value.replace(/\D/g, ""));
  }

  async function handlePay() {
    if (amount < MIN_AMOUNT || submitting) return;
    setError(null);
    setSubmitting(true);
    try {
      const { payment_link } = await initiatePurchase(amount);
      // The callback page needs to know the balance as it was *before* this
      // payment, so it can detect "did it actually increase" reliably - if
      // it instead captured that baseline from its own first poll, a fast
      // webhook (landing before that first poll) would corrupt the baseline
      // to already equal the new balance, making the increase-check never
      // fire and forcing a full ~15s timeout fallback even though crediting
      // had already succeeded.
      if (balanceKobo !== null) {
        sessionStorage.setItem("theresa:pre-payment-balance-kobo", String(balanceKobo));
      }
      window.location.href = payment_link;
    } catch {
      setError("Couldn't start payment - please try again.");
      setSubmitting(false);
    }
  }

  return (
    <AppShell>
      <div className="mx-auto max-w-lg px-6 py-10">
        <Link
          href="/dashboard"
          className="mb-6 inline-flex items-center gap-1.5 text-sm text-[var(--color-text-secondary)] transition-colors hover:text-[var(--color-text-primary)]"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to dashboard
        </Link>

        <Card className="p-7 text-center">
          <div className="flex justify-center">
            <Pill>Voice credits</Pill>
          </div>
          <p className="mt-4 text-4xl font-bold text-[var(--color-text-primary)]">
            {balanceKobo !== null ? formatNaira(balanceKobo) : "…"}
          </p>
          <p className="mt-1.5 text-sm text-[var(--color-text-secondary)]">
            {freeTrialSecondsLeft > 0
              ? `+ ${Math.ceil(freeTrialSecondsLeft / 60)} min of free voice remaining`
              : "current balance"}
          </p>
        </Card>

        <Card className="mt-6 p-5">
          <p className="text-sm font-semibold text-[var(--color-text-primary)]">Choose an amount</p>

          <div className="mt-3 grid grid-cols-2 gap-3">
            {PRESET_AMOUNTS.map((preset) => {
              const active = !customAmount && selected === preset;
              return (
                <button
                  key={preset}
                  type="button"
                  onClick={() => {
                    setSelected(preset);
                    setCustomAmount("");
                  }}
                  className={`relative rounded-[var(--radius-md)] border px-4 py-3.5 text-left transition-all ${
                    active
                      ? "border-[var(--color-accent)] bg-[var(--color-accent)]/10 shadow-[var(--shadow-xs)]"
                      : "border-[var(--color-border-subtle)] hover:border-[var(--color-border)] hover:bg-[var(--color-surface-hover)]"
                  }`}
                >
                  <span
                    className={`text-base font-bold ${
                      active ? "text-[var(--color-accent)]" : "text-[var(--color-text-primary)]"
                    }`}
                  >
                    ₦ {preset.toLocaleString()}
                  </span>
                  {active && (
                    <span className="absolute right-2.5 top-2.5 flex h-4 w-4 items-center justify-center rounded-full bg-[var(--color-accent)] text-white">
                      <Check className="h-2.5 w-2.5" strokeWidth={3} />
                    </span>
                  )}
                </button>
              );
            })}
          </div>

          <div className="mt-4">
            <label className="mb-1.5 block text-xs font-medium text-[var(--color-text-secondary)]">
              Or enter a custom amount (min ₦ {MIN_AMOUNT.toLocaleString()})
            </label>
            <div className="relative">
              <span className="pointer-events-none absolute left-3.5 top-1/2 -translate-y-1/2 text-sm font-medium text-[var(--color-text-secondary)]">
                ₦
              </span>
              <input
                type="text"
                inputMode="numeric"
                value={customAmount ? Number(customAmount).toLocaleString("en-NG") : ""}
                onChange={handleCustomAmountChange}
                placeholder="3,000"
                className="w-full rounded-[var(--radius-md)] border border-[var(--color-border-subtle)] bg-[var(--color-surface)] py-2.5 pl-7 pr-3 text-sm text-[var(--color-text-primary)] outline-none transition-colors focus:border-[var(--color-accent)]"
              />
            </div>
          </div>

          {error && (
            <p className="mt-4 flex items-center gap-1.5 rounded-[var(--radius-md)] bg-[var(--color-danger)]/10 px-3 py-2 text-xs text-[var(--color-danger)]">
              <AlertCircle className="h-3.5 w-3.5 shrink-0" />
              {error}
            </p>
          )}

          <button
            type="button"
            onClick={handlePay}
            disabled={amount < MIN_AMOUNT || submitting}
            className="mt-5 flex w-full items-center justify-center rounded-[var(--radius-full)] bg-[var(--color-accent)] px-4 py-3 text-sm font-semibold text-[var(--color-accent-foreground)] shadow-[var(--shadow-sm)] transition-opacity hover:opacity-90 disabled:opacity-60"
          >
            {submitting ? "Processing…" : `Pay ₦ ${amount.toLocaleString()}`}
          </button>

          <p className="mt-3 flex items-center justify-center gap-1.5 text-xs text-[var(--color-text-secondary)]">
            <ShieldCheck className="h-3.5 w-3.5" />
            Secured by Flutterwave · credits never expire
          </p>
        </Card>
      </div>
    </AppShell>
  );
}
