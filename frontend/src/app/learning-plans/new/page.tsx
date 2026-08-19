"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { GraduationCap } from "lucide-react";
import { apiFetch } from "@/lib/api";
import { AppShell } from "@/components/layout/AppShell";
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
        <div className="flex flex-col items-center text-center">
          <div className="flex h-14 w-14 items-center justify-center rounded-[var(--radius-full)] bg-[var(--color-accent)]/10 text-[var(--color-accent)]">
            <GraduationCap className="h-7 w-7" />
          </div>
          <h1 className="mt-4 text-3xl font-bold tracking-tight text-[var(--color-text-primary)]">
            Create my lesson plan
          </h1>
          <p className="mt-2 max-w-md text-sm text-[var(--color-text-secondary)]">
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
