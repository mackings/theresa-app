import Link from "next/link";
import { ArrowRight, Check, FileUp, Mic, Sparkles } from "lucide-react";
import { ThemeToggle } from "@/components/theme/ThemeToggle";
import { Pill } from "@/components/ui/Pill";
import { PhoneMockup } from "@/components/ui/PhoneMockup";
import { FeatureShowcaseCard } from "@/components/ui/FeatureShowcaseCard";

const CHECKLIST = [
  "Teaches any topic, not just uploaded material",
  "Real-time voice explanations, Pidgin-inflected",
  "A live board that writes itself as she teaches",
];

function BoardPreview() {
  return (
    <div className="w-full space-y-2 rounded-[var(--radius-md)] bg-[var(--color-surface-raised)] p-4 shadow-[var(--shadow-xs)]">
      <div className="h-2.5 w-4/5 rounded-full bg-[var(--color-border)]" />
      <div className="h-2.5 w-3/5 rounded-full bg-[var(--color-border)]" />
      <div className="flex items-center gap-1">
        <div className="h-2.5 w-2/5 rounded-full bg-[var(--color-accent)]/40" />
        <div className="h-3 w-[2px] animate-pulse bg-[var(--color-accent)]" />
      </div>
    </div>
  );
}

function VoicePreview({ size = "h-16 w-16" }: { size?: string }) {
  return (
    <div
      className={`${size} rounded-full`}
      style={{
        background:
          "radial-gradient(circle at 32% 28%, color-mix(in srgb, var(--color-accent) 35%, white), var(--color-accent) 60%)",
        boxShadow: "0 8px 24px color-mix(in srgb, var(--color-accent) 35%, transparent)",
      }}
    />
  );
}

function UploadPreview() {
  return (
    <div className="flex w-full flex-col items-center gap-2 rounded-[var(--radius-md)] border border-dashed border-[var(--color-border)] bg-[var(--color-surface-raised)] px-4 py-5">
      <FileUp className="h-6 w-6 text-[var(--color-accent)]" />
      <p className="text-xs text-[var(--color-text-secondary)]">lecture-notes.pdf</p>
    </div>
  );
}

const FEATURES = [
  {
    preview: <BoardPreview />,
    title: "Watch it get written",
    description: "A real board, typed out step by step, in sync with the explanation.",
  },
  {
    preview: <VoicePreview />,
    title: "Hear it explained",
    description: "A warm, patient voice that teaches out loud, Pidgin-inflected.",
  },
  {
    preview: <UploadPreview />,
    title: "Upload anything",
    description: "A PDF, a photo of a page, or a pasted problem, understood in seconds.",
  },
];

const CONVERSATION = [
  { role: "user" as const, text: "Can you explain how photosynthesis works?" },
  { role: "assistant" as const, kind: "board" as const, stepCount: 5 },
  { role: "user" as const, text: "Can you just say that out loud instead?" },
  { role: "assistant" as const, kind: "voice" as const },
];

export default function Home() {
  return (
    <div className="flex min-h-full flex-col bg-[var(--color-bg)]">
      <header className="flex items-center justify-between px-4 py-3 sm:px-6 sm:py-4">
        <div className="flex shrink-0 items-center gap-2">
          <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-[var(--radius-md)] bg-[var(--color-accent)] text-sm font-bold text-[var(--color-accent-foreground)] sm:h-8 sm:w-8 sm:text-base">
            T
          </div>
          <span className="text-sm font-semibold text-[var(--color-text-primary)]">
            Theresa
          </span>
        </div>
        <div className="flex shrink-0 items-center gap-1.5 sm:gap-3">
          <ThemeToggle />
          <Link
            href="/login"
            className="shrink-0 whitespace-nowrap rounded-[var(--radius-sm)] px-2.5 py-1.5 text-xs font-medium text-[var(--color-text-primary)] transition-colors hover:bg-[var(--color-surface-hover)] sm:px-4 sm:py-2 sm:text-sm"
          >
            Log in
          </Link>
          <Link
            href="/signup"
            className="flex shrink-0 items-center gap-1.5 whitespace-nowrap rounded-[var(--radius-md)] bg-[var(--color-accent)] px-3 py-1.5 text-xs font-semibold text-[var(--color-accent-foreground)] shadow-[var(--shadow-xs)] transition-opacity hover:opacity-90 sm:px-4 sm:py-2 sm:text-sm"
          >
            Get Started
            <ArrowRight className="hidden h-3.5 w-3.5 sm:inline-block" />
          </Link>
        </div>
      </header>

      <main className="flex-1">
        <section className="relative overflow-hidden">
          <div
            aria-hidden
            className="pointer-events-none absolute -left-32 -top-32 h-80 w-80 rounded-full opacity-30 blur-3xl"
            style={{ background: "var(--color-voice)" }}
          />
          <div
            aria-hidden
            className="pointer-events-none absolute -right-24 top-10 h-72 w-72 rounded-full opacity-20 blur-3xl"
            style={{ background: "var(--color-accent)" }}
          />

          <div className="relative mx-auto grid w-full max-w-6xl grid-cols-1 items-center gap-12 px-6 py-14 lg:grid-cols-2 lg:py-24">
            <div>
              <Pill icon={<Sparkles className="h-3.5 w-3.5 text-[var(--color-accent)]" />}>
                AI-powered tutoring
              </Pill>

              <h1 className="mt-5 text-4xl font-bold leading-[1.1] tracking-tight text-[var(--color-text-primary)] sm:text-5xl lg:text-6xl">
                Learn anything,
                <br />
                <span className="text-[var(--color-accent)]">taught out loud.</span>
              </h1>
              <p className="mt-5 max-w-lg text-base text-[var(--color-text-secondary)] sm:text-lg">
                Ask a question, paste a problem, or upload your notes. Theresa
                explains it with a real voice while writing it out on a live
                board, like a tutor beside you, not a search result.
              </p>

              <div className="mt-7 flex flex-col gap-3 sm:flex-row sm:items-center">
                <Link
                  href="/signup"
                  className="flex items-center justify-center gap-2 rounded-[var(--radius-md)] bg-[var(--color-accent)] px-6 py-3 text-sm font-semibold text-[var(--color-accent-foreground)] shadow-[var(--shadow-sm)] transition-opacity hover:opacity-90"
                >
                  Get Started
                  <ArrowRight className="h-4 w-4" />
                </Link>
                <Link
                  href="/login"
                  className="rounded-[var(--radius-md)] border border-[var(--color-border)] px-6 py-3 text-center text-sm font-semibold text-[var(--color-text-primary)] transition-colors hover:bg-[var(--color-surface-hover)]"
                >
                  Log in
                </Link>
              </div>

              <ul className="mt-8 space-y-2.5">
                {CHECKLIST.map((item) => (
                  <li key={item} className="flex items-center gap-2.5 text-sm text-[var(--color-text-secondary)]">
                    <span className="flex h-[18px] w-[18px] shrink-0 items-center justify-center rounded-full bg-[var(--color-accent)]/15 text-[var(--color-accent)]">
                      <Check className="h-3 w-3" />
                    </span>
                    {item}
                  </li>
                ))}
              </ul>
            </div>

            <div className="relative mx-auto flex w-full max-w-sm items-center justify-center py-8 lg:max-w-none">
              <svg
                aria-hidden
                className="pointer-events-none absolute left-0 top-0 hidden h-full w-full lg:block"
                viewBox="0 0 528 351"
                preserveAspectRatio="none"
                fill="none"
              >
                <path
                  d="M 70 92 C 95 118, 112 72, 140 48"
                  stroke="var(--color-accent)"
                  strokeWidth="1.5"
                  strokeDasharray="3 5"
                  opacity="0.5"
                  vectorEffect="non-scaling-stroke"
                />
                <circle cx="140" cy="48" r="6" fill="var(--color-accent)" />
                <circle cx="140" cy="48" r="6" fill="var(--color-accent)" opacity="0.4">
                  <animate attributeName="r" values="6;14;6" dur="2.2s" repeatCount="indefinite" />
                  <animate attributeName="opacity" values="0.4;0;0.4" dur="2.2s" repeatCount="indefinite" />
                </circle>
              </svg>

              <div className="absolute left-2 top-6 hidden lg:block">
                <div className="flex h-20 w-20 items-center justify-center rounded-full border-4 border-[var(--color-surface-raised)] bg-[var(--color-surface-raised)] shadow-[var(--shadow-lg)]">
                  <VoicePreview size="h-14 w-14" />
                </div>
              </div>

              <PhoneMockup>
                <div className="space-y-3 px-4 pb-6">
                  <div className="flex items-center justify-between">
                    <p className="text-sm font-semibold text-[var(--color-text-primary)]">
                      Good morning
                    </p>
                  </div>
                  <div
                    className="rounded-[var(--radius-lg)] bg-[var(--color-accent)] p-4 text-white shadow-[var(--shadow-md)]"
                  >
                    <p className="text-xs uppercase tracking-wide text-white/70">Board</p>
                    <p className="mt-1 text-sm font-medium">Newton&apos;s Second Law</p>
                    <p className="mt-2 text-lg font-semibold">F = m · a</p>
                  </div>
                  <div className="space-y-1.5 rounded-[var(--radius-md)] bg-[var(--color-surface)] p-3">
                    <div className="h-2 w-4/5 rounded-full bg-[var(--color-border)]" />
                    <div className="h-2 w-3/5 rounded-full bg-[var(--color-border)]" />
                  </div>
                </div>
              </PhoneMockup>

              <div className="absolute -bottom-4 -left-4 hidden w-52 rounded-[var(--radius-lg)] border border-[var(--color-border-subtle)] bg-[var(--color-surface-raised)] p-3 shadow-[var(--shadow-lg)] lg:block">
                <div className="flex items-center gap-1.5">
                  <div className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-[var(--color-accent)]/15">
                    <Sparkles className="h-2.5 w-2.5 text-[var(--color-accent)]" />
                  </div>
                  <p className="text-xs font-medium text-[var(--color-text-primary)]">Theresa</p>
                </div>
                <p className="mt-1.5 text-xs text-[var(--color-text-secondary)]">
                  Explained on the board · 6 steps
                </p>
              </div>

              <div className="absolute right-0 top-2 hidden sm:block">
                <Pill icon={<Check className="h-3.5 w-3.5 text-[var(--color-accent)]" />}>
                  Private &amp; secure
                </Pill>
              </div>
              <div className="absolute bottom-16 right-0 hidden sm:block lg:-right-4">
                <Pill icon={<Mic className="h-3.5 w-3.5 text-[var(--color-accent)]" />}>
                  Real-time voice
                </Pill>
              </div>
            </div>
          </div>
        </section>

        <section className="mx-auto w-full max-w-3xl px-6 py-16 sm:py-20">
          <div className="text-center">
            <Pill icon={<Sparkles className="h-3.5 w-3.5 text-[var(--color-accent)]" />}>
              Real conversations
            </Pill>
            <h2 className="mt-4 text-3xl font-bold tracking-tight text-[var(--color-text-primary)] sm:text-4xl">
              Ask anything. Theresa actually answers.
            </h2>
            <p className="mx-auto mt-3 max-w-lg text-[var(--color-text-secondary)]">
              A real back-and-forth, switching between voice and text
              whenever you want.
            </p>
          </div>

          <div className="mt-10 space-y-4 rounded-[var(--radius-lg)] border border-[var(--color-border-subtle)] bg-[var(--color-surface-raised)] p-4 shadow-[var(--shadow-md)] sm:p-6">
            {CONVERSATION.map((turn, i) =>
              turn.role === "user" ? (
                <div
                  key={i}
                  style={{ animationDelay: `${i * 150}ms` }}
                  className="fade-in-up ml-auto max-w-[85%] rounded-[var(--radius-lg)] bg-[var(--color-accent)] px-3.5 py-2 text-sm text-[var(--color-accent-foreground)] shadow-[var(--shadow-xs)]"
                >
                  {turn.text}
                </div>
              ) : (
                <div
                  key={i}
                  style={{ animationDelay: `${i * 150}ms` }}
                  className="fade-in-up mr-auto flex max-w-[85%] items-center gap-2"
                >
                  <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-[var(--color-accent)] text-[var(--color-accent-foreground)]">
                    {turn.kind === "voice" ? (
                      <Mic className="h-3.5 w-3.5" />
                    ) : (
                      <Sparkles className="h-3.5 w-3.5" />
                    )}
                  </div>
                  <div className="rounded-[var(--radius-lg)] border border-[var(--color-border-subtle)] bg-[var(--color-surface)] px-3.5 py-2 text-sm text-[var(--color-text-secondary)]">
                    {turn.kind === "voice"
                      ? "Switched to voice · listening"
                      : `Explained on the board · ${turn.stepCount} steps`}
                  </div>
                </div>
              )
            )}
          </div>
        </section>

        <section className="bg-[var(--color-surface)] px-6 py-16 sm:py-20">
          <div className="mx-auto w-full max-w-6xl">
            <div className="text-center">
              <h2 className="text-3xl font-bold tracking-tight text-[var(--color-text-primary)] sm:text-4xl">
                What Makes Theresa Different
              </h2>
              <p className="mx-auto mt-3 max-w-xl text-[var(--color-text-secondary)]">
                Real teaching, built to help you actually understand, not just
                skim a summary.
              </p>
            </div>

            <div className="mt-10 grid grid-cols-1 gap-6 sm:grid-cols-3">
              {FEATURES.map((feature, i) => (
                <FeatureShowcaseCard
                  key={feature.title}
                  {...feature}
                  style={{ animationDelay: `${i * 80}ms` }}
                  className="fade-in-up transition-all hover:-translate-y-0.5 hover:shadow-[var(--shadow-md)]"
                />
              ))}
            </div>
          </div>
        </section>

        <section className="relative overflow-hidden bg-[var(--color-accent)] px-6 py-16 text-[var(--color-accent-foreground)] sm:py-20">
          <div
            aria-hidden
            className="pointer-events-none absolute -right-20 -top-20 h-64 w-64 rounded-full bg-white/10 blur-3xl"
          />
          <div
            aria-hidden
            className="pointer-events-none absolute -bottom-24 -left-16 h-64 w-64 rounded-full bg-white/10 blur-3xl"
          />
          <div className="relative mx-auto flex max-w-4xl flex-col items-center gap-6 text-center">
            <h2 className="text-3xl font-bold leading-tight tracking-tight sm:text-4xl">
              Ready to actually understand it?
            </h2>
            <Link
              href="/signup"
              className="flex items-center gap-2 rounded-[var(--radius-md)] bg-white px-6 py-3 text-sm font-semibold text-[var(--color-accent)] shadow-[var(--shadow-sm)] transition-opacity hover:opacity-90"
            >
              Get Started
              <ArrowRight className="h-4 w-4" />
            </Link>
          </div>
        </section>
      </main>

      <footer className="border-t border-[var(--color-border)] px-6 py-8 text-xs text-[var(--color-text-secondary)]">
        <div className="mx-auto flex w-full max-w-6xl flex-col items-center gap-4 sm:flex-row sm:justify-between">
          <div className="flex flex-wrap items-center justify-center gap-2">
            <div className="flex h-5 w-5 items-center justify-center rounded-[var(--radius-sm)] bg-[var(--color-accent)] text-[10px] font-bold text-[var(--color-accent-foreground)]">
              T
            </div>
            <span>Theresa</span>
            <span className="text-[var(--color-border)]">·</span>
            <span className="opacity-70">A product of Decode Analytical</span>
          </div>
          <span>&copy; {new Date().getFullYear()} Theresa. All rights reserved.</span>
        </div>
      </footer>
    </div>
  );
}
