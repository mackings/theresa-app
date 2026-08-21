import type { MetadataRoute } from "next";

// Auth-gated app routes have no real content for an anonymous crawler (they
// just redirect to /login) - excluded to keep crawl budget on the pages that
// actually have something to index.
export default function robots(): MetadataRoute.Robots {
  return {
    rules: {
      userAgent: "*",
      allow: "/",
      disallow: ["/dashboard", "/session", "/learning-plans", "/credits", "/profile", "/payment", "/api"],
    },
    sitemap: "https://asktheresa.com/sitemap.xml",
  };
}
