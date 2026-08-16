"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { NotebookPen, Sparkles } from "lucide-react";
import { apiFetch } from "@/lib/api";
import { AppShell } from "@/components/layout/AppShell";
import { Pill } from "@/components/ui/Pill";
import { AccountTypeGate } from "@/features/learning-plans/components/AccountTypeGate";
import { PlanForm } from "@/features/learning-plans/components/PlanForm";

type Me = { account_type?: "personal" | "organization" | "" };

export default function NewLearningPlanPage() {
  const router = useRouter();
  const [me, setMe] = useState<Me | null>(null);

  useEffect(() => {
    apiFetch<Me>("/api/auth/me")
      .then(setMe)
      .catch(() => router.replace("/login"));
  }, [router]);

  if (!me) return null;

  return (
    <AppShell>
      <div className="mx-auto max-w-2xl space-y-8 px-6 py-10">
        <div className="text-center">
          <Pill icon={<Sparkles className="h-3 w-3" />}>New plan</Pill>
          <h1 className="mt-3 flex items-center justify-center gap-2 text-3xl font-bold tracking-tight text-[var(--color-text-primary)]">
            <NotebookPen className="h-6 w-6 text-[var(--color-accent)]" />
            Create my lesson plan
          </h1>
          <p className="mt-2 text-sm text-[var(--color-text-secondary)]">
            Tell Theresa what you want to learn and how much time you have - she&rsquo;ll build
            a day-by-day plan and teach it with you, one step at a time.
          </p>
        </div>

        {!me.account_type ? (
          <AccountTypeGate onDone={(accountType) => setMe({ ...me, account_type: accountType })} />
        ) : (
          <PlanForm />
        )}
      </div>
    </AppShell>
  );
}
