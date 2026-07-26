// A plain module-level flag, not React state - Sidebar and VoiceControls are
// siblings under AppShell (Sidebar never renders the session content), so
// there's no prop/context path between them. Sidebar only ever needs to read
// this once, synchronously, at the moment of a navigation click - it doesn't
// need to re-render when it changes, so a shared mutable value is simpler
// than wiring up a Context just for this one read.
let active = false;

export function setActiveVoiceSession(value: boolean) {
  active = value;
}

export function isActiveVoiceSession(): boolean {
  return active;
}
