import { ImageResponse } from "next/og";

export const alt = "Theresa - AI Tutor That Teaches on a Live Board";
export const size = { width: 1200, height: 630 };
export const contentType = "image/png";

export default function Image() {
  return new ImageResponse(
    (
      <div
        style={{
          width: "100%",
          height: "100%",
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
          justifyContent: "center",
          background: "linear-gradient(135deg, #1f4640 0%, #2f6459 55%, #3e8573 100%)",
          color: "#ffffff",
          fontFamily: "sans-serif",
        }}
      >
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            width: 120,
            height: 120,
            borderRadius: "50%",
            background: "rgba(255,255,255,0.15)",
            fontSize: 64,
            fontWeight: 700,
            marginBottom: 32,
          }}
        >
          T
        </div>
        <div style={{ fontSize: 76, fontWeight: 700, letterSpacing: -1 }}>Theresa</div>
        <div style={{ fontSize: 32, marginTop: 20, color: "rgba(255,255,255,0.85)" }}>
          The AI tutor that teaches on a live board
        </div>
      </div>
    ),
    { ...size }
  );
}
