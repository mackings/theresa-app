"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { GraduationCap, NotebookPen, Plus } from "lucide-react";
import { apiFetch } from "@/lib/api";
import { AppShell } from "@/components/layout/AppShell";
import { Card } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { AccountTypeGate } from "@/features/learning-plans/components/AccountTypeGate";
import { PlanCard } from "@/features/learning-plans/components/PlanCard";
import { ContinueLearning } from "@/features/learning-plans/components/ContinueLearning";
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
          <div className="flex items-center gap-3">
            <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-[var(--radius-lg)] bg-[var(--color-accent)]/10 text-[var(--color-accent)]">
              <GraduationCap className="h-5 w-5" />
            </div>
            <div>
              <h1 className="text-2xl font-bold tracking-tight text-[var(--color-text-primary)]">
                Learning plans
              </h1>
              <p className="text-sm text-[var(--color-text-secondary)]">
                Paced, day-by-day plans Theresa builds and teaches from.
              </p>
            </div>
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

        {plans !== null && plans.length > 0 && <ContinueLearning plans={plans} />}

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
          <div>
            <p className="mb-3 text-xs font-medium uppercase tracking-wide text-[var(--color-text-secondary)]">
              All plans
            </p>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {plans.map((plan) => (
                <PlanCard
                  key={plan.id}
                  plan={plan}
                  onClick={() => router.push(`/learning-plans/${plan.id}`)}
                />
              ))}
            </div>
          </div>
        )}
      </div>
    </AppShell>
  );
}
