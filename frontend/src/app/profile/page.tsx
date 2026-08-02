"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import {
  ArrowLeft,
  ArrowDownCircle,
  ArrowUpCircle,
  CalendarDays,
  Gift,
  LogOut,
  Mail,
  Receipt,
  Settings as SettingsIcon,
  ShieldAlert,
  ShieldCheck,
} from "lucide-react";
import { apiFetch } from "@/lib/api";
import { AppShell } from "@/components/layout/AppShell";
import { Card, CardHeader } from "@/components/ui/Card";
import { Avatar } from "@/components/ui/Avatar";
import { ThemeToggle } from "@/components/theme/ThemeToggle";
import { ConfirmDialog } from "@/components/ui/ConfirmDialog";
import { getTransactions, formatNaira, CreditTransaction } from "@/lib/credits";

type Me = {
  id: string;
  email: string;
  name: string;
  email_verified: boolean;
  created_at: string;
};

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("en-US", {
    year: "numeric",
    month: "long",
    day: "numeric",
  });
}

function transactionLabel(tx: CreditTransaction): string {
  switch (tx.type) {
    case "purchase":
      return "Credit purchase";
    case "voice_usage":
      return "Voice session usage";
    case "free_trial":
      return "Free trial credit";
    default:
      return tx.type;
  }
}

function TransactionIcon({ type }: { type: CreditTransaction["type"] }) {
  if (type === "voice_usage") {
    return <ArrowDownCircle className="h-4 w-4 text-[var(--color-text-secondary)]" />;
  }
  if (type === "free_trial") {
    return <Gift className="h-4 w-4 text-[var(--color-voice)]" />;
  }
  return <ArrowUpCircle className="h-4 w-4 text-[var(--color-accent)]" />;
}

export default function ProfilePage() {
  const router = useRouter();
  const [me, setMe] = useState<Me | null>(null);
  const [transactions, setTransactions] = useState<CreditTransaction[]>([]);
  const [loadingTx, setLoadingTx] = useState(true);
  const [loggingOut, setLoggingOut] = useState(false);
  const [confirmingLogout, setConfirmingLogout] = useState(false);

  useEffect(() => {
    apiFetch<Me>("/api/auth/me")
      .then(setMe)
      .catch(() => router.replace("/login"));
    getTransactions()
      .then(setTransactions)
      .catch(() => {})
      .finally(() => setLoadingTx(false));
  }, [router]);

  async function handleLogout() {
    if (loggingOut) return;
    setLoggingOut(true);
    try {
      await apiFetch("/api/auth/logout", { method: "POST" });
    } catch {
      // The cookie is httpOnly and can't be cleared client-side either way -
      // send the user to /login regardless so they're never stuck on a page
      // that thinks they're signed in when the server might disagree.
    } finally {
      router.push("/login");
    }
  }

  return (
    <AppShell>
      <div className="mx-auto max-w-2xl px-6 py-10">
        <Link
          href="/dashboard"
          className="mb-6 inline-flex items-center gap-1.5 text-sm text-[var(--color-text-secondary)] transition-colors hover:text-[var(--color-text-primary)]"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to dashboard
        </Link>

        <Card className="p-6">
          <div className="flex items-center gap-4">
            <Avatar name={me?.name ?? "?"} size={56} />
            <div className="min-w-0">
              <p className="truncate text-lg font-semibold text-[var(--color-text-primary)]">
                {me?.name ?? "…"}
              </p>
              <p className="flex items-center gap-1.5 text-sm text-[var(--color-text-secondary)]">
                <Mail className="h-3.5 w-3.5 shrink-0" />
                <span className="truncate">{me?.email ?? ""}</span>
              </p>
            </div>
          </div>
          <div className="mt-4 flex flex-wrap items-center gap-2 text-xs text-[var(--color-text-secondary)]">
            {me && (
              <span
                className={`inline-flex items-center gap-1 rounded-[var(--radius-full)] px-2.5 py-1 ${
                  me.email_verified
                    ? "bg-[var(--color-accent)]/10 text-[var(--color-accent)]"
                    : "bg-[var(--color-danger)]/10 text-[var(--color-danger)]"
                }`}
              >
                {me.email_verified ? (
                  <ShieldCheck className="h-3.5 w-3.5" />
                ) : (
                  <ShieldAlert className="h-3.5 w-3.5" />
                )}
                {me.email_verified ? "Email verified" : "Email not verified"}
              </span>
            )}
            {me?.created_at && (
              <span className="inline-flex items-center gap-1">
                <CalendarDays className="h-3.5 w-3.5" />
                Member since {formatDate(me.created_at)}
              </span>
            )}
          </div>
        </Card>

        <Card className="mt-6">
          <CardHeader icon={<SettingsIcon className="h-3.5 w-3.5" />} title="Settings" />
          <div className="flex items-center justify-between px-4 py-4">
            <div>
              <p className="text-sm font-medium text-[var(--color-text-primary)]">Appearance</p>
              <p className="text-xs text-[var(--color-text-secondary)]">
                Choose light or dark mode
              </p>
            </div>
            <ThemeToggle />
          </div>
        </Card>

        <Card className="mt-6">
          <CardHeader icon={<Receipt className="h-3.5 w-3.5" />} title="Transactions" />
          <div className="divide-y divide-[var(--color-border-subtle)] px-4 pb-1 pt-2">
            {loadingTx && (
              <p className="py-4 text-sm text-[var(--color-text-secondary)]">Loading…</p>
            )}
            {!loadingTx && transactions.length === 0 && (
              <p className="py-4 text-sm text-[var(--color-text-secondary)]">
                No transactions yet.
              </p>
            )}
            {transactions.map((tx) => (
              <div key={tx.id} className="flex items-center justify-between gap-3 py-3">
                <div className="flex min-w-0 items-center gap-2.5">
                  <TransactionIcon type={tx.type} />
                  <div className="min-w-0">
                    <p className="truncate text-sm text-[var(--color-text-primary)]">
                      {transactionLabel(tx)}
                    </p>
                    <p className="text-xs text-[var(--color-text-secondary)]">
                      {formatDate(tx.created_at)}
                    </p>
                  </div>
                </div>
                <p
                  className={`shrink-0 text-sm font-semibold ${
                    tx.amount_kobo < 0
                      ? "text-[var(--color-text-secondary)]"
                      : "text-[var(--color-accent)]"
                  }`}
                >
                  {tx.amount_kobo >= 0 ? "+" : "-"}
                  {formatNaira(Math.abs(tx.amount_kobo))}
                </p>
              </div>
            ))}
          </div>
        </Card>

        <button
          type="button"
          onClick={() => setConfirmingLogout(true)}
          className="mt-6 flex w-full items-center justify-center gap-2 rounded-[var(--radius-lg)] border border-[var(--color-border-subtle)] bg-[var(--color-surface-raised)] px-4 py-3 text-sm font-medium text-[var(--color-danger)] shadow-[var(--shadow-xs)] transition-colors hover:bg-[var(--color-surface-hover)]"
        >
          <LogOut className="h-4 w-4" />
          Log out
        </button>
      </div>

      <ConfirmDialog
        open={confirmingLogout}
        title="Log out?"
        description="You'll need to log in again to access your sessions."
        confirmLabel={loggingOut ? "Logging out…" : "Log out"}
        cancelLabel="Cancel"
        onConfirm={handleLogout}
        onCancel={() => setConfirmingLogout(false)}
      />
    </AppShell>
  );
}
