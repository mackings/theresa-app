// A deterministic "subject identity" color for a learning plan - same plan
// always gets the same color (hashed from its id, not random per render),
// so a course reads as a consistent visual identity across the list page,
// the continue-learning cards, and its own detail page, the same way a
// course thumbnail color stays fixed on Khan Academy/Coursera-style course
// cards rather than reshuffling on every visit.
export interface SubjectPalette {
  bg: string;
  // Full-strength background (no opacity) - for things like a solid accent
  // strip, where a washed/translucent tone wouldn't read as a real color.
  // Kept as its own literal class (not derived from `bg` at runtime) since
  // Tailwind only generates CSS for class strings it can find verbatim in
  // source - a computed/`.replace()`'d class name never gets generated.
  solid: string;
  text: string;
  ring: string;
}

const PALETTE: SubjectPalette[] = [
  { bg: "bg-[var(--color-accent)]/12", solid: "bg-[var(--color-accent)]", text: "text-[var(--color-accent)]", ring: "var(--color-accent)" },
  { bg: "bg-[var(--color-subject-blue)]/12", solid: "bg-[var(--color-subject-blue)]", text: "text-[var(--color-subject-blue)]", ring: "var(--color-subject-blue)" },
  { bg: "bg-[var(--color-subject-purple)]/12", solid: "bg-[var(--color-subject-purple)]", text: "text-[var(--color-subject-purple)]", ring: "var(--color-subject-purple)" },
  { bg: "bg-[var(--color-subject-orange)]/12", solid: "bg-[var(--color-subject-orange)]", text: "text-[var(--color-subject-orange)]", ring: "var(--color-subject-orange)" },
  { bg: "bg-[var(--color-subject-pink)]/12", solid: "bg-[var(--color-subject-pink)]", text: "text-[var(--color-subject-pink)]", ring: "var(--color-subject-pink)" },
  { bg: "bg-[var(--color-subject-amber)]/12", solid: "bg-[var(--color-subject-amber)]", text: "text-[var(--color-subject-amber)]", ring: "var(--color-subject-amber)" },
  { bg: "bg-[var(--color-voice)]/15", solid: "bg-[var(--color-voice)]", text: "text-[var(--color-voice)]", ring: "var(--color-voice)" },
];

export function subjectPaletteFor(id: string): SubjectPalette {
  let hash = 0;
  for (let i = 0; i < id.length; i++) hash = (hash * 31 + id.charCodeAt(i)) >>> 0;
  return PALETTE[hash % PALETTE.length];
}
