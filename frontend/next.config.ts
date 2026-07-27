import type { NextConfig } from "next";

// Both forms are needed in connect-src: fetch/streaming uses http(s), the
// voice feature's WebSocket uses ws(s) - derived from the same env var
// ws-client.ts itself derives the WS URL from (NEXT_PUBLIC_API_URL, http-\>ws).
const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8090";
const apiWsUrl = apiUrl.replace(/^http/, "ws");

// Tried pinning script-src to a hash of just layout.tsx's inline anti-FOUC
// theme script, but live-tested that against a real production build and
// it broke the app: Next.js's App Router itself injects several other
// inline scripts (hydration/RSC payload data) with their own per-build
// hashes that can't be predicted or pinned in advance, so a hash-only
// script-src blocked React from hydrating at all. 'unsafe-inline' here is a
// deliberate, tested tradeoff - the real XSS surfaces in this app (KaTeX
// math, Mermaid diagrams) are already safely configured independent of CSP
// (trust:false, securityLevel:strict), so this isn't leaving the primary
// defense down, just accepting that a strict script-src isn't practical
// with this framework without a per-request nonce system.
// React's dev-mode debugging tools (stack-trace reconstruction) use eval(),
// which the real production build never does (React's own console message
// confirms this) - 'unsafe-eval' is scoped to non-production only so local
// `next dev` isn't noisy, without weakening the actual deployed CSP.
const isDev = process.env.NODE_ENV !== "production";

const csp = [
  "default-src 'self'",
  `script-src 'self' 'unsafe-inline'${isDev ? " 'unsafe-eval'" : ""}`,
  "style-src 'self' 'unsafe-inline'",
  "img-src 'self' data:",
  "font-src 'self' data:",
  `connect-src 'self' ${apiUrl} ${apiWsUrl}`,
  "frame-ancestors 'none'",
  "base-uri 'self'",
  "object-src 'none'",
  "form-action 'self'",
].join("; ");

const nextConfig: NextConfig = {
  async headers() {
    return [
      {
        source: "/:path*",
        headers: [
          { key: "X-Content-Type-Options", value: "nosniff" },
          { key: "X-Frame-Options", value: "DENY" },
          { key: "Referrer-Policy", value: "no-referrer" },
          { key: "Permissions-Policy", value: "camera=(), geolocation=(), payment=()" },
          { key: "Content-Security-Policy", value: csp },
        ],
      },
    ];
  },
};

export default nextConfig;
