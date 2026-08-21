import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";

const inter = Inter({
  variable: "--font-inter",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  metadataBase: new URL("https://asktheresa.com"),
  title: {
    default: "Theresa - AI Tutor That Teaches on a Live Board",
    template: "%s | Theresa",
  },
  description:
    "Theresa is an AI tutor that teaches your course material step by step on a live, visual board - upload a PDF or ask a question, and learn by voice or text, with real 3D anatomy, code practice, paced learning plans, and quizzes along the way.",
  keywords: [
    "AI tutor",
    "AI tutoring",
    "learn with AI",
    "online tutor",
    "study with AI",
    "AI teacher",
    "voice tutor",
    "learning plan generator",
    "PDF to lesson",
  ],
  authors: [{ name: "Decode Analytical" }],
  creator: "Decode Analytical",
  publisher: "Decode Analytical",
  category: "education",
  alternates: {
    canonical: "/",
  },
  openGraph: {
    type: "website",
    url: "https://asktheresa.com",
    siteName: "Theresa",
    title: "Theresa - AI Tutor That Teaches on a Live Board",
    description:
      "An AI tutor that teaches your course material step by step on a live, visual board - by voice or text.",
    locale: "en_US",
  },
  twitter: {
    card: "summary_large_image",
    title: "Theresa - AI Tutor That Teaches on a Live Board",
    description:
      "An AI tutor that teaches your course material step by step on a live, visual board - by voice or text.",
  },
  robots: {
    index: true,
    follow: true,
    googleBot: {
      index: true,
      follow: true,
      "max-image-preview": "large",
    },
  },
};

const themeInitScript = `
(function () {
  try {
    var stored = localStorage.getItem("theresa-theme");
    if (stored === "light" || stored === "dark") {
      document.documentElement.setAttribute("data-theme", stored);
    }
  } catch (e) {}
})();
`;

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className={`${inter.variable} h-full antialiased`}>
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeInitScript }} />
      </head>
      <body className="h-full font-sans" suppressHydrationWarning>
        {children}
      </body>
    </html>
  );
}
