"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { NotebookPen, Plus, Sparkles } from "lucide-react";
import { apiFetch } from "@/lib/api";
import { AppShell } from "@/components/layout/AppShell";
import { Card } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { Pill } from "@/components/ui/Pill";
import { AccountTypeGate } from "@/features/learning-plans/components/AccountTypeGate";
import { PlanCard } from "@/features/learning-plans/components/PlanCard";
import { listLearningPlans } from "@/features/learning-plans/lib/api";
import { LearningPlan } from "@/features/learning-plans/types";

type Me = { account_type?: "personal" | "organization" | "" };

export default function LearningPlansPage() {
  const router = useRouter();
  const [me, setMe] = useState<Me | null>(null);
  const [plans, setPlans] = useState<LearningPlan[] | null>(null);

  useEffect(() => {
    apiFetch<Me>("/api/auth/me")
      .then(setMe)
      .catch(() => router.replace("/login"));
    listLearningPlans()
      .then(setPlans)
      .catch(() => setPlans([]));
  }, [router]);

  if (!me) return null;

  return (
    <AppShell>
      <div className="mx-auto max-w-4xl space-y-8 px-6 py-10">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <Pill icon={<Sparkles className="h-3 w-3" />}>Paced by Theresa</Pill>
            <h1 className="mt-3 flex items-center gap-2 text-3xl font-bold tracking-tight text-[var(--color-text-primary)]">
              <NotebookPen className="h-6 w-6 text-[var(--color-accent)]" />
              Learning plans
            </h1>
          </div>
          <Button
            variant="primary"
            icon={<Plus className="h-4 w-4" />}
            onClick={() => router.push("/learning-plans/new")}
          >
            New plan
          </Button>
        </div>

        {!me.account_type && (
          <AccountTypeGate onDone={(accountType) => setMe({ ...me, account_type: accountType })} />
        )}

        {plans === null && (
          <p className="text-sm text-[var(--color-text-secondary)]">Loading…</p>
        )}

        {plans !== null && plans.length === 0 && (
          <Card className="flex flex-col items-center gap-3 px-6 py-12 text-center">
            <div className="flex h-12 w-12 items-center justify-center rounded-[var(--radius-full)] bg-[var(--color-accent)]/10 text-[var(--color-accent)]">
              <NotebookPen className="h-5 w-5" />
            </div>
            <p className="text-sm text-[var(--color-text-secondary)]">
              No learning plans yet - create one to get a paced plan Theresa builds and
              teaches from.
            </p>
            <Button
              variant="secondary"
              icon={<Plus className="h-4 w-4" />}
              onClick={() => router.push("/learning-plans/new")}
            >
              Create your first plan
            </Button>
          </Card>
        )}

        {plans !== null && plans.length > 0 && (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {plans.map((plan) => (
              <PlanCard
                key={plan.id}
                plan={plan}
                onClick={() => router.push(`/learning-plans/${plan.id}`)}
              />
            ))}
          </div>
        )}
      </div>
    </AppShell>
  );
}
