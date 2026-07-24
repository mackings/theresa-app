"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import {
  Calculator,
  GraduationCap,
  MessageCircleQuestion,
  MessageSquare,
  MessageSquarePlus,
  Mic,
  Send,
  Sparkles,
  X,
} from "lucide-react";
import { apiFetch } from "@/lib/api";
import { createSession } from "@/lib/sessions";
import { AppShell } from "@/components/layout/AppShell";
import { Card } from "@/components/ui/Card";
import { FeatureShowcaseCard } from "@/components/ui/FeatureShowcaseCard";
import { IconButton } from "@/components/ui/Button";
import { SpeakingOrb } from "@/components/voice/SpeakingOrb";
import { UploadDropzone } from "@/components/chat/UploadDropzone";
import { TutorSession, DocumentMeta } from "@/types/board";

type Me = { id: string; email: string; name: string };

function greeting(): string {
  const hour = new Date().getHours();
  if (hour < 12) return "Good morning";
  if (hour < 18) return "Good afternoon";
  return "Good evening";
}

function relativeTime(iso: string): string {
  const diffMs = Date.now() - new Date(iso).getTime();
  const minutes = Math.round(diffMs / 60000);
  if (minutes < 1) return "just now";
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.round(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.round(hours / 24);
  if (days < 7) return `${days}d ago`;
  return new Date(iso).toLocaleDateString();
}

const ACTIONS = [
  {
    icon: GraduationCap,
    label: "Teach my course",
    action: "upload" as const,
    style: "tint" as const,
  },
  {
    icon: Calculator,
    label: "Paste or type a problem",
    action: "focus" as const,
    style: "solid" as const,
  },
  {
    icon: MessageCircleQuestion,
    label: "Ask anything",
    action: "focus" as const,
    style: "outline" as const,
  },
];

export default function DashboardPage() {
  const router = useRouter();
  const [me, setMe] = useState<Me | null>(null);
  const [sessions, setSessions] = useState<TutorSession[] | null>(null);
  const [creating, setCreating] = useState(false);
  const [prompt, setPrompt] = useState("");
  const [showUpload, setShowUpload] = useState(false);
  const promptInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    apiFetch<Me>("/api/auth/me")
      .then(setMe)
      .catch(() => router.replace("/login"));
    apiFetch<TutorSession[]>("/api/sessions")
      .then(setSessions)
      .catch(() => setSessions([]));
  }, [router]);

  async function handleCreateSession(mode: "text" | "voice") {
    if (creating) return;
    setCreating(true);
    try {
      const session = await createSession(mode);
      router.push(`/session/${session.id}`);
    } finally {
      setCreating(false);
    }
  }

  async function handleAskSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!prompt.trim() || creating) return;
    setCreating(true);
    try {
      const session = await createSession("text");
      sessionStorage.setItem(
        `theresa:pending-message:${session.id}`,
        JSON.stringify({ text: prompt })
      );
      router.push(`/session/${session.id}`);
    } finally {
      setCreating(false);
    }
  }

  function handleActionClick(action: "upload" | "focus") {
    if (action === "upload") {
      setShowUpload(true);
    } else {
      promptInputRef.current?.focus();
    }
  }

  async function handleDocumentReady(doc: DocumentMeta) {
    if (creating) return;
    setCreating(true);
    try {
      const session = await createSession("text");
      sessionStorage.setItem(
        `theresa:pending-message:${session.id}`,
        JSON.stringify({ text: "Teach me this material", documentId: doc.id })
      );
      router.push(`/session/${session.id}`);
    } finally {
      setCreating(false);
    }
  }

  if (!me) {
    return null;
  }

  return (
    <AppShell>
      <div className="mx-auto max-w-4xl space-y-10 px-6 py-10">
        <div className="flex flex-col items-center text-center">
          <SpeakingOrb state="listening" size={88} />
          <p className="mt-4 text-sm text-[var(--color-text-secondary)]">
            {greeting()}, {me.name}
          </p>
          <h1 className="mt-1 text-3xl font-bold tracking-tight text-[var(--color-text-primary)] sm:text-4xl">
            How can I help you <span className="text-[var(--color-accent)]">today</span>?
          </h1>
        </div>

        <form
          onSubmit={handleAskSubmit}
          className="rounded-[var(--radius-lg)] border border-[var(--color-border-subtle)] bg-[var(--color-surface-raised)] p-3 shadow-[var(--shadow-md)]"
        >
          <input
            ref={promptInputRef}
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            placeholder="Ask me anything..."
            className="w-full bg-transparent px-2 py-2 text-sm text-[var(--color-text-primary)] outline-none"
          />
          <div className="mt-1 flex items-center justify-between px-2">
            <span className="text-xs text-[var(--color-text-secondary)]">
              Press enter to start a new session
            </span>
            <IconButton
              type="submit"
              variant="primary"
              aria-label="Ask Theresa"
              disabled={!prompt.trim() || creating}
              className="h-9 w-9 rounded-[var(--radius-full)]"
            >
              <Send className="h-4 w-4" />
            </IconButton>
          </div>
        </form>

        {showUpload ? (
          <div className="rounded-[var(--radius-lg)] border border-[var(--color-border-subtle)] bg-[var(--color-surface-raised)] shadow-[var(--shadow-sm)]">
            <div className="flex items-center justify-between px-4 pt-3">
              <p className="text-xs font-medium uppercase tracking-wide text-[var(--color-text-secondary)]">
                Teach my course
              </p>
              <IconButton
                variant="ghost"
                aria-label="Cancel upload"
                onClick={() => setShowUpload(false)}
                className="h-7 w-7"
              >
                <X className="h-3.5 w-3.5" />
              </IconButton>
            </div>
            <UploadDropzone onDocumentReady={handleDocumentReady} />
          </div>
        ) : (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            {ACTIONS.map((a, i) => {
              const badgeClass =
                a.style === "solid"
                  ? "bg-[var(--color-accent)] text-white"
                  : a.style === "outline"
                    ? "border border-[var(--color-accent)] text-[var(--color-accent)]"
                    : "bg-[var(--color-accent)]/10 text-[var(--color-accent)]";
              return (
                <button
                  key={a.label}
                  type="button"
                  onClick={() => handleActionClick(a.action)}
                  style={{ animationDelay: `${i * 60}ms` }}
                  className="fade-in-up flex items-start gap-3 rounded-[var(--radius-lg)] border border-[var(--color-border-subtle)] bg-[var(--color-surface-raised)] p-4 text-left shadow-[var(--shadow-xs)] transition-all hover:-translate-y-0.5 hover:shadow-[var(--shadow-md)]"
                >
                  <div className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-[var(--radius-md)] ${badgeClass}`}>
                    <a.icon className="h-4 w-4" />
                  </div>
                  <p className="text-sm font-medium text-[var(--color-text-primary)]">{a.label}</p>
                </button>
              );
            })}
          </div>
        )}

        <div>
          <p className="mb-3 text-xs font-medium uppercase tracking-wide text-[var(--color-text-secondary)]">
            Quick start
          </p>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <FeatureShowcaseCard
              onClick={() => handleCreateSession("text")}
              className={`cursor-pointer transition-all hover:-translate-y-0.5 hover:shadow-[var(--shadow-md)] ${creating ? "pointer-events-none opacity-60" : ""}`}
              title="New text session"
              description="Paste a problem or upload course material"
              preview={
                <div className="w-full max-w-[220px] space-y-2 rounded-[var(--radius-md)] bg-[var(--color-surface-raised)] p-4 shadow-[var(--shadow-xs)]">
                  <div className="h-2.5 w-4/5 rounded-full bg-[var(--color-border)]" />
                  <div className="h-2.5 w-3/5 rounded-full bg-[var(--color-border)]" />
                  <div className="flex items-center gap-2">
                    <MessageSquarePlus className="h-4 w-4 text-[var(--color-accent)]" />
                    <div className="h-2.5 w-2/5 rounded-full bg-[var(--color-accent)]/40" />
                  </div>
                </div>
              }
            />
            <FeatureShowcaseCard
              onClick={() => handleCreateSession("voice")}
              className={`cursor-pointer transition-all hover:-translate-y-0.5 hover:shadow-[var(--shadow-md)] ${creating ? "pointer-events-none opacity-60" : ""}`}
              title="New voice session"
              description="Talk it through with Theresa, live"
              preview={<SpeakingOrb state="speaking" size={64} />}
            />
          </div>
        </div>

        <div>
          <p className="mb-3 text-xs font-medium uppercase tracking-wide text-[var(--color-text-secondary)]">
            Recent sessions
          </p>

          {sessions === null && (
            <p className="text-sm text-[var(--color-text-secondary)]">Loading…</p>
          )}

          {sessions !== null && sessions.length === 0 && (
            <Card className="flex flex-col items-center gap-2 px-6 py-10 text-center">
              <Sparkles className="h-6 w-6 text-[var(--color-text-secondary)]" />
              <p className="text-sm text-[var(--color-text-secondary)]">
                No sessions yet — start one above to see it here.
              </p>
            </Card>
          )}

          {sessions !== null && sessions.length > 0 && (
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {sessions.map((session, i) => {
                const isVoice = session.mode === "voice";
                const ModeIcon = isVoice ? Mic : MessageSquare;
                return (
                  <Card
                    key={session.id}
                    onClick={() => router.push(`/session/${session.id}`)}
                    style={{ animationDelay: `${Math.min(i, 8) * 40}ms` }}
                    className="fade-in-up cursor-pointer p-4 transition-all hover:-translate-y-0.5 hover:shadow-[var(--shadow-md)]"
                  >
                    <div className="flex items-center justify-between">
                      <span
                        className={`flex items-center gap-1 rounded-[var(--radius-full)] px-2 py-0.5 text-[11px] font-medium uppercase tracking-wide ${
                          isVoice
                            ? "bg-[var(--color-voice)]/15 text-[var(--color-voice)]"
                            : "bg-[var(--color-accent)]/10 text-[var(--color-accent)]"
                        }`}
                      >
                        <ModeIcon className="h-3 w-3" />
                        {isVoice ? "Voice" : "Text"}
                      </span>
                    </div>
                    <p className="mt-3 truncate text-sm font-semibold text-[var(--color-text-primary)]">
                      {session.title}
                    </p>
                    <p className="mt-1 text-xs text-[var(--color-text-secondary)]">
                      {relativeTime(session.updated_at)}
                    </p>
                  </Card>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </AppShell>
  );
}
