"use client";

import Image from "next/image";
import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { ThemeToggle } from "@/components/theme/ThemeToggle";
import { DemoExperience } from "@/components/demo/DemoExperience";

// The public, no-signup preview - see backend's DemoHandler for what "demo
// account" actually means (a real, ephemeral, password-less user with a
// tiny free-trial allowance). DemoExperience does the actual work (minting
// the account, opening the voice session, rendering Board + VoiceControls);
// this page is just its header chrome.
export default function TryPage() {
  return (
    <div className="flex h-full flex-col bg-[var(--color-bg)]">
      <header className="flex shrink-0 items-center justify-between border-b border-[var(--color-border)] px-4 py-2.5 sm:px-6">
        <Link href="/" className="flex shrink-0 items-center gap-2">
          <Image
            src="/brand/theresa-face.png"
            alt="Theresa"
            width={28}
            height={28}
            className="h-7 w-7 shrink-0 rounded-full object-cover"
          />
          <span className="text-sm font-semibold text-[var(--color-text-primary)]">Theresa</span>
        </Link>
        <div className="flex shrink-0 items-center gap-1.5 sm:gap-3">
          <ThemeToggle />
          <Link
            href="/signup"
            className="flex shrink-0 items-center gap-1.5 whitespace-nowrap rounded-[var(--radius-md)] bg-[var(--color-accent)] px-3 py-1.5 text-xs font-semibold text-[var(--color-accent-foreground)] shadow-[var(--shadow-xs)] transition-opacity hover:opacity-90 sm:px-4 sm:py-2 sm:text-sm"
          >
            Sign up to save progress
            <ArrowRight className="hidden h-3.5 w-3.5 sm:inline-block" />
          </Link>
        </div>
      </header>

      <DemoExperience />
    </div>
  );
}
