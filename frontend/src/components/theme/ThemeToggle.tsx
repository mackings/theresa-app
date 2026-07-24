"use client";

import { useSyncExternalStore } from "react";
import { Sun, Moon } from "lucide-react";

type Theme = "light" | "dark";

const THEME_STORAGE_KEY = "theresa-theme";
const THEME_CHANGE_EVENT = "theresa-theme-change";

function readTheme(): Theme {
  const stored = window.localStorage.getItem(THEME_STORAGE_KEY);
  if (stored === "light" || stored === "dark") return stored;
  return window.matchMedia("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
}

function getServerSnapshot(): Theme {
  return "light";
}

function subscribe(callback: () => void) {
  const media = window.matchMedia("(prefers-color-scheme: dark)");
  media.addEventListener("change", callback);
  window.addEventListener(THEME_CHANGE_EVENT, callback);
  return () => {
    media.removeEventListener("change", callback);
    window.removeEventListener(THEME_CHANGE_EVENT, callback);
  };
}

function applyTheme(theme: Theme) {
  document.documentElement.setAttribute("data-theme", theme);
  window.localStorage.setItem(THEME_STORAGE_KEY, theme);
  window.dispatchEvent(new Event(THEME_CHANGE_EVENT));
}

export function ThemeToggle() {
  const theme = useSyncExternalStore(subscribe, readTheme, getServerSnapshot);

  return (
    <div
      role="group"
      aria-label="Color theme"
      className="flex items-center gap-0.5 rounded-[var(--radius-full)] border border-[var(--color-border)] bg-[var(--color-surface-raised)] p-0.5"
    >
      <button
        type="button"
        onClick={() => applyTheme("light")}
        aria-label="Light theme"
        aria-pressed={theme === "light"}
        className={`flex h-6 w-6 items-center justify-center rounded-[var(--radius-full)] transition-colors ${
          theme === "light"
            ? "bg-[var(--color-surface-hover)] text-[var(--color-text-primary)] shadow-[var(--shadow-xs)]"
            : "text-[var(--color-text-secondary)]"
        }`}
      >
        <Sun className="h-3.5 w-3.5" />
      </button>
      <button
        type="button"
        onClick={() => applyTheme("dark")}
        aria-label="Dark theme"
        aria-pressed={theme === "dark"}
        className={`flex h-6 w-6 items-center justify-center rounded-[var(--radius-full)] transition-colors ${
          theme === "dark"
            ? "bg-[var(--color-surface-hover)] text-[var(--color-text-primary)] shadow-[var(--shadow-xs)]"
            : "text-[var(--color-text-secondary)]"
        }`}
      >
        <Moon className="h-3.5 w-3.5" />
      </button>
    </div>
  );
}
