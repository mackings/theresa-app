"use client";

import { useState } from "react";
import { GraduationCap, Users } from "lucide-react";
import { apiFetch } from "@/lib/api";
import { Card } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";

// Shown once, before a user's first access to the learning-plans feature,
// whenever their account has no account_type set yet - a pure demand-signal
// flag (see backend/internal/models/user.go's AccountType doc comment), not
// a gate that gains or loses any functionality either way. Once set,
// onDone's caller should never render this again for that user.
export function AccountTypeGate({
  onDone,
}: {
  onDone: (accountType: "personal" | "organization") => void;
}) {
  const [saving, setSaving] = useState<"personal" | "organization" | null>(null);

  async function choose(accountType: "personal" | "organization") {
    if (saving) return;
    setSaving(accountType);
    try {
      await apiFetch("/api/auth/account-type", {
        method: "POST",
        body: JSON.stringify({ account_type: accountType }),
      });
      onDone(accountType);
    } finally {
      setSaving(null);
    }
  }

  return (
    <Card className="p-5">
      <p className="text-sm font-medium text-[var(--color-text-primary)]">
        One quick question before you get started
      </p>
      <p className="mt-1 text-sm text-[var(--color-text-secondary)]">
        Are you using Theresa for yourself, or for an organization teaching multiple
        students?
      </p>
      <div className="mt-4 flex flex-col gap-2 sm:flex-row">
        <Button
          variant="secondary"
          icon={<GraduationCap className="h-4 w-4" />}
          disabled={saving !== null}
          onClick={() => choose("personal")}
          className="flex-1"
        >
          Just for myself
        </Button>
        <Button
          variant="secondary"
          icon={<Users className="h-4 w-4" />}
          disabled={saving !== null}
          onClick={() => choose("organization")}
          className="flex-1"
        >
          For an organization
        </Button>
      </div>
    </Card>
  );
}
