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

function VoicePreview() {
  return (
    <div
      className="h-16 w-16 rounded-full"
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
    description: "A PDF, a photo of a page, or a pasted problem — understood in seconds.",
  },
];

export default function Home() {
  return (
    <div className="flex min-h-full flex-col bg-[var(--color-bg)]">
      <header className="flex items-center justify-between px-6 py-4">
        <div className="flex items-center gap-2">
          <div className="flex h-8 w-8 items-center justify-center rounded-[var(--radius-md)] bg-[var(--color-accent)] text-base font-bold text-[var(--color-accent-foreground)]">
            T
          </div>
          <span className="text-sm font-semibold text-[var(--color-text-primary)]">
            Theresa
          </span>
        </div>
        <div className="flex items-center gap-3">
          <ThemeToggle />
          <Link
            href="/login"
            className="rounded-[var(--radius-sm)] px-4 py-2 text-sm font-medium text-[var(--color-text-primary)] transition-colors hover:bg-[var(--color-surface-hover)]"
          >
            Log in
          </Link>
          <Link
            href="/signup"
            className="flex items-center gap-1.5 rounded-[var(--radius-md)] bg-[var(--color-accent)] px-4 py-2 text-sm font-semibold text-[var(--color-accent-foreground)] shadow-[var(--shadow-xs)] transition-opacity hover:opacity-90"
          >
            Get Started
            <ArrowRight className="h-3.5 w-3.5" />
          </Link>
        </div>
      </header>

      <main className="flex-1">
        <section className="mx-auto grid w-full max-w-6xl grid-cols-1 items-center gap-12 px-6 py-16 lg:grid-cols-2 lg:py-24">
          <div>
            <Pill icon={<Sparkles className="h-3.5 w-3.5 text-[var(--color-accent)]" />}>
              AI-powered tutoring
            </Pill>

            <h1 className="mt-5 text-5xl font-bold leading-[1.08] tracking-tight text-[var(--color-text-primary)] sm:text-6xl">
              Taught out loud,
              <br />
              <span className="text-[var(--color-accent)]">by Theresa.</span>
            </h1>
            <p className="mt-5 max-w-lg text-lg text-[var(--color-text-secondary)]">
              Ask a question, paste a problem, or upload your notes — Theresa
              teaches any topic on a live board, out loud.
            </p>

            <div className="mt-7 flex items-center gap-3">
              <Link
                href="/signup"
                className="flex items-center gap-2 rounded-[var(--radius-md)] bg-[var(--color-accent)] px-6 py-3 text-sm font-semibold text-[var(--color-accent-foreground)] shadow-[var(--shadow-sm)] transition-opacity hover:opacity-90"
              >
                Get Started
                <ArrowRight className="h-4 w-4" />
              </Link>
              <Link
                href="/login"
                className="rounded-[var(--radius-md)] border border-[var(--color-border)] px-6 py-3 text-sm font-semibold text-[var(--color-text-primary)] transition-colors hover:bg-[var(--color-surface-hover)]"
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

          <div className="relative flex items-center justify-center py-8">
            <PhoneMockup>
              <div className="space-y-3 px-4 pb-6">
                <div className="flex items-center justify-between">
                  <p className="text-sm font-semibold text-[var(--color-text-primary)]">
                    Good morning
                  </p>
                </div>
                <div
                  className="rounded-[var(--radius-lg)] p-4 text-white shadow-[var(--shadow-md)]"
                  style={{ background: "linear-gradient(135deg, var(--color-accent), color-mix(in srgb, var(--color-accent) 70%, black))" }}
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

            <div className="absolute left-0 top-6 hidden sm:block">
              <Pill icon={<Check className="h-3.5 w-3.5 text-[var(--color-accent)]" />}>
                Private &amp; secure
              </Pill>
            </div>
            <div className="absolute bottom-10 right-0 hidden sm:block">
              <Pill icon={<Mic className="h-3.5 w-3.5 text-[var(--color-accent)]" />}>
                Real-time voice
              </Pill>
            </div>
          </div>
        </section>

        <section className="mx-auto w-full max-w-6xl px-6 py-16">
          <div className="text-center">
            <h2 className="text-3xl font-bold tracking-tight text-[var(--color-text-primary)] sm:text-4xl">
              What Makes Theresa Different
            </h2>
            <p className="mx-auto mt-3 max-w-xl text-[var(--color-text-secondary)]">
              Real teaching, built to help you actually understand — not just
              skim a summary.
            </p>
          </div>

          <div className="mt-10 grid grid-cols-1 gap-6 sm:grid-cols-3">
            {FEATURES.map((feature) => (
              <FeatureShowcaseCard key={feature.title} {...feature} />
            ))}
          </div>
        </section>

        <section className="bg-[var(--color-accent)] px-6 py-20 text-[var(--color-accent-foreground)]">
          <div className="mx-auto flex max-w-4xl flex-col items-center gap-6 text-center">
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

      <footer className="flex items-center justify-between border-t border-[var(--color-border)] px-6 py-6 text-xs text-[var(--color-text-secondary)]">
        <div className="flex items-center gap-2">
          <div className="flex h-5 w-5 items-center justify-center rounded-[var(--radius-sm)] bg-[var(--color-accent)] text-[10px] font-bold text-[var(--color-accent-foreground)]">
            T
          </div>
          <span>Theresa</span>
        </div>
        <span>&copy; {new Date().getFullYear()} Theresa. All rights reserved.</span>
      </footer>
    </div>
  );
}
