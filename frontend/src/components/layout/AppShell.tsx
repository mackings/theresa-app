"use client";

import { useEffect, useState } from "react";
import { PanelLeft } from "lucide-react";
import { Sidebar } from "@/components/layout/Sidebar";

const MOBILE_BREAKPOINT = 1024;

export function AppShell({ children }: { children: React.ReactNode }) {
  const [sidebarOpen, setSidebarOpen] = useState(true);

  useEffect(() => {
    // Viewport width is unknowable during SSR, so the default (open) is only
    // ever corrected client-side, once, right after mount - not a recurring
    // sync, so the one-time setState here is the deliberate exception.
    if (window.innerWidth < MOBILE_BREAKPOINT) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setSidebarOpen(false);
    }
  }, []);

  return (
    <div className="relative flex h-full overflow-hidden bg-[var(--color-bg)]">
      {sidebarOpen && (
        <>
          <div
            aria-hidden
            onClick={() => setSidebarOpen(false)}
            className="fixed inset-0 z-30 bg-black/30 lg:hidden"
          />
          <div className="fixed inset-y-0 left-0 z-40 lg:static lg:z-auto">
            <Sidebar onClose={() => setSidebarOpen(false)} />
          </div>
        </>
      )}

      <div className="relative flex flex-1 flex-col overflow-hidden">
        {!sidebarOpen && (
          <button
            type="button"
            onClick={() => setSidebarOpen(true)}
            aria-label="Open sidebar"
            className="absolute left-3 top-3 z-20 flex h-9 w-9 items-center justify-center rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface-raised)] text-[var(--color-text-secondary)] shadow-[var(--shadow-xs)] transition-colors hover:bg-[var(--color-surface-hover)] hover:text-[var(--color-text-primary)]"
          >
            <PanelLeft className="h-4 w-4" />
          </button>
        )}
        <main className="flex-1 overflow-y-auto">{children}</main>
      </div>
    </div>
  );
}
