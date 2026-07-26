"use client";

import { useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import { MessageSquarePlus, Mic, MessageSquare, PanelLeftClose, Plus, Sparkles } from "lucide-react";
import { ThemeToggle } from "@/components/theme/ThemeToggle";
import { Button, IconButton } from "@/components/ui/Button";
import { Avatar } from "@/components/ui/Avatar";
import { ConfirmDialog } from "@/components/ui/ConfirmDialog";
import { apiFetch } from "@/lib/api";
import { createSession } from "@/lib/sessions";
import { getCreditBalance } from "@/lib/credits";
import { isActiveVoiceSession } from "@/lib/activeVoiceSession";
import { TutorSession } from "@/types/board";

const MOBILE_BREAKPOINT = 1024;

type Me = { id: string; email: string; name: string };

function startOfDay(d: Date): number {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
}

function groupByRecency(sessions: TutorSession[]): { label: string; sessions: TutorSession[] }[] {
  const today = startOfDay(new Date());
  const yesterday = today - 86400000;

  const groups: Record<"Today" | "Yesterday" | "Older", TutorSession[]> = {
    Today: [],
    Yesterday: [],
    Older: [],
  };

  for (const session of sessions) {
    const t = startOfDay(new Date(session.updated_at));
    if (t === today) groups.Today.push(session);
    else if (t === yesterday) groups.Yesterday.push(session);
    else groups.Older.push(session);
  }

  return (["Today", "Yesterday", "Older"] as const)
    .filter((label) => groups[label].length > 0)
    .map((label) => ({ label, sessions: groups[label] }));
}

export function Sidebar({ onClose }: { onClose?: () => void }) {
  const router = useRouter();
  const pathname = usePathname();
  const [sessions, setSessions] = useState<TutorSession[]>([]);
  const [me, setMe] = useState<Me | null>(null);
  const [creating, setCreating] = useState(false);
  const [freeTrialSecondsLeft, setFreeTrialSecondsLeft] = useState<number | null>(null);
  const [percentRemaining, setPercentRemaining] = useState<number | null>(null);
  const [pendingNav, setPendingNav] = useState<(() => void) | null>(null);

  useEffect(() => {
    apiFetch<TutorSession[]>("/api/sessions")
      .then(setSessions)
      .catch(() => {});
    apiFetch<Me>("/api/auth/me")
      .then(setMe)
      .catch(() => {});
    getCreditBalance()
      .then((b) => {
        setFreeTrialSecondsLeft(b.free_trial_seconds_remaining);
        setPercentRemaining(b.percent_remaining);
      })
      .catch(() => {});
  }, []);

  function closeOnMobile() {
    if (window.innerWidth < MOBILE_BREAKPOINT) onClose?.();
  }

  // Navigating away from an active voice session drops the call - if one's
  // in progress, confirm first instead of silently ending it out from under
  // the user the moment they click anything else in the sidebar.
  function guardedNavigate(action: () => void) {
    if (isActiveVoiceSession()) {
      setPendingNav(() => action);
      return;
    }
    action();
  }

  async function handleCreateSession(mode: "text" | "voice") {
    if (creating) return;
    setCreating(true);
    try {
      const session = await createSession(mode);
      closeOnMobile();
      router.push(`/session/${session.id}`);
    } catch {
      router.push("/login");
    } finally {
      setCreating(false);
    }
  }

  const groups = groupByRecency(sessions);

  return (
    <aside className="flex h-full w-64 shrink-0 flex-col border-r border-[var(--color-border)] bg-[var(--color-surface)]">
      <div className="flex items-center justify-between gap-2 px-4 py-4">
        <div className="flex items-center gap-2">
          <div className="flex h-7 w-7 items-center justify-center rounded-[var(--radius-sm)] bg-[var(--color-accent)] text-sm font-bold text-[var(--color-accent-foreground)]">
            T
          </div>
          <span className="text-sm font-semibold text-[var(--color-text-primary)]">
            Theresa
          </span>
        </div>
        {onClose && (
          <IconButton
            variant="ghost"
            aria-label="Close sidebar"
            onClick={onClose}
            className="h-7 w-7"
          >
            <PanelLeftClose className="h-4 w-4" />
          </IconButton>
        )}
      </div>

      <div className="space-y-1.5 px-3">
        <Button
          variant="secondary"
          onClick={() => guardedNavigate(() => handleCreateSession("text"))}
          disabled={creating}
          icon={<MessageSquarePlus className="h-4 w-4" />}
          className="w-full justify-start"
        >
          New session
        </Button>
        <Button
          variant="secondary"
          onClick={() => guardedNavigate(() => handleCreateSession("voice"))}
          disabled={creating}
          icon={<Mic className="h-4 w-4" />}
          className="w-full justify-start"
        >
          Voice session
        </Button>
      </div>

      <nav className="mt-4 flex-1 space-y-3 overflow-y-auto px-3 pb-3">
        {groups.map((group) => (
          <div key={group.label}>
            <p className="px-2 pb-1 text-xs font-medium text-[var(--color-text-secondary)]">
              {group.label}
            </p>
            <div className="space-y-0.5">
              {group.sessions.map((session) => {
                const active = pathname === `/session/${session.id}`;
                const ModeIcon = session.mode === "voice" ? Mic : MessageSquare;
                return (
                  <button
                    key={session.id}
                    type="button"
                    onClick={() =>
                      guardedNavigate(() => {
                        closeOnMobile();
                        router.push(`/session/${session.id}`);
                      })
                    }
                    className={`flex w-full items-center gap-2 truncate rounded-[var(--radius-md)] px-2 py-2 text-left text-sm transition-colors ${
                      active
                        ? "bg-[var(--color-surface-hover)] text-[var(--color-text-primary)]"
                        : "text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-hover)] hover:text-[var(--color-text-primary)]"
                    }`}
                  >
                    <ModeIcon className="h-3.5 w-3.5 shrink-0" />
                    <span className="truncate">{session.title}</span>
                  </button>
                );
              })}
            </div>
          </div>
        ))}
      </nav>

      {percentRemaining !== null && (
        <button
          type="button"
          onClick={() =>
            guardedNavigate(() => {
              closeOnMobile();
              router.push("/credits");
            })
          }
          className="flex items-center justify-between border-t border-[var(--color-border)] px-3 py-2.5 text-left transition-colors hover:bg-[var(--color-surface-hover)]"
        >
          <span className="flex items-center gap-1.5 text-xs text-[var(--color-text-secondary)]">
            <Sparkles className="h-3.5 w-3.5 text-[var(--color-accent)]" />
            {freeTrialSecondsLeft && freeTrialSecondsLeft > 0
              ? `${Math.ceil(freeTrialSecondsLeft / 60)} min free trial left`
              : `${percentRemaining}% usage remaining`}
          </span>
          <span className="flex items-center gap-1 rounded-[var(--radius-full)] bg-[var(--color-accent)]/10 px-2 py-1 text-xs font-medium text-[var(--color-accent)]">
            <Plus className="h-3 w-3" />
            Add
          </span>
        </button>
      )}

      <div className="flex items-center justify-between border-t border-[var(--color-border)] px-3 py-3">
        <div className="flex min-w-0 items-center gap-2 text-sm text-[var(--color-text-primary)]">
          <Avatar name={me?.name ?? "?"} size={26} />
          <span className="truncate">{me?.name ?? "Guest"}</span>
        </div>
        <ThemeToggle />
      </div>

      <ConfirmDialog
        open={pendingNav !== null}
        title="End this voice session?"
        description="You're in an active voice conversation - leaving now will end the call."
        confirmLabel="End session"
        cancelLabel="Stay"
        onConfirm={() => {
          const action = pendingNav;
          setPendingNav(null);
          action?.();
        }}
        onCancel={() => setPendingNav(null)}
      />
    </aside>
  );
}
