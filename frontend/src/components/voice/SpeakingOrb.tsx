export type OrbState = "idle" | "listening" | "speaking" | "reconnecting";

const stateColor: Record<OrbState, string> = {
  idle: "var(--color-text-secondary)",
  listening: "var(--color-voice)",
  speaking: "var(--color-accent)",
  reconnecting: "var(--color-text-secondary)",
};

// A glossy gradient sphere, not a flat circle: layered radial-gradients fake
// a highlight + a pooled shadow over a color wash, plus a soft blurred glow -
// pure CSS (no three.js/WebGL, same constraint the 3D hand avatar was
// descoped under). Purely a decorative status display, driven by a derived
// data-state - the actual mic toggle is a separate button in VoiceControls,
// matching the reference where the big sphere isn't itself the click target.
export function SpeakingOrb({ state, size = 144 }: { state: OrbState; size?: number }) {
  const color = stateColor[state];
  const orbColorVar = { "--orb-color": color } as React.CSSProperties;
  const breatheDuration = state === "speaking" ? "1.1s" : "2.6s";

  return (
    <div
      data-state={state}
      className="relative flex items-center justify-center"
      style={{ ...orbColorVar, width: size, height: size }}
    >
      {state === "listening" && <span className="orb-ring" />}
      {state === "speaking" && (
        <>
          <span className="orb-ring" style={{ animationDuration: "1.1s" }} />
          <span
            className="orb-ring"
            style={{ animationDuration: "1.1s", animationDelay: "0.4s" }}
          />
        </>
      )}
      {state === "reconnecting" && (
        <span
          className="absolute inset-0 rounded-full border-2 border-dashed"
          style={{ borderColor: color, animation: "spin-dash 1.4s linear infinite" }}
        />
      )}

      <span
        className="relative h-full w-full rounded-full"
        style={{
          background: [
            "radial-gradient(circle at 30% 24%, rgba(255,255,255,0.95) 0%, transparent 12%)",
            "radial-gradient(circle at 32% 28%, rgba(255,255,255,0.7) 0%, transparent 42%)",
            "radial-gradient(circle at 72% 78%, rgba(0,0,0,0.28) 0%, transparent 55%)",
            `radial-gradient(circle at 42% 40%, color-mix(in srgb, ${color} 55%, white), color-mix(in srgb, ${color} 70%, black) 100%)`,
          ].join(", "),
          boxShadow: `0 12px 32px color-mix(in srgb, ${color} 35%, transparent)`,
          animation:
            state === "idle"
              ? undefined
              : `sphere-breathe ${breatheDuration} ease-in-out infinite`,
          opacity: state === "idle" ? 0.5 : 1,
          filter: state === "idle" ? "saturate(0.4)" : undefined,
        }}
      />
    </div>
  );
}
