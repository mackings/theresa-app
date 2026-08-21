import type { MetadataRoute } from "next";

// Only the genuinely public, indexable routes - everything else under
// /dashboard, /session, /learning-plans etc. is auth-gated and has no
// content for an anonymous visitor or crawler (see robots.ts).
export default function sitemap(): MetadataRoute.Sitemap {
  const base = "https://asktheresa.com";
  return [
    { url: base, lastModified: new Date(), changeFrequency: "weekly", priority: 1 },
    { url: `${base}/login`, lastModified: new Date(), changeFrequency: "monthly", priority: 0.5 },
    { url: `${base}/signup`, lastModified: new Date(), changeFrequency: "monthly", priority: 0.8 },
  ];
}
