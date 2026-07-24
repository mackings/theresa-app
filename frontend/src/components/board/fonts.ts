import { Caveat } from "next/font/google";

// Scoped to the Board subtree only (not loaded in the root layout) - the
// handwriting font has no use anywhere else in the app.
export const caveat = Caveat({
  variable: "--font-caveat",
  subsets: ["latin"],
});
