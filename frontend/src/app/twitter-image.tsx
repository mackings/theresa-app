import { ImageResponse } from "next/og";
import { join } from "node:path";
import { readFile } from "node:fs/promises";

export const alt = "Theresa - AI Tutor That Teaches on a Live Board";
export const size = { width: 1200, height: 630 };
export const contentType = "image/png";

export default async function Image() {
  const avatarData = await readFile(join(process.cwd(), "public/brand/theresa-avatar.png"), "base64");
  const avatarSrc = `data:image/png;base64,${avatarData}`;

  return new ImageResponse(
    (
      <div
        style={{
          width: "100%",
          height: "100%",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          gap: 48,
          background: "linear-gradient(135deg, #1f4640 0%, #2f6459 55%, #3e8573 100%)",
          color: "#ffffff",
          fontFamily: "sans-serif",
        }}
      >
        <img src={avatarSrc} alt="" width={420} height={420} style={{ objectFit: "contain" }} />
        <div style={{ display: "flex", flexDirection: "column" }}>
          <div style={{ fontSize: 76, fontWeight: 700, letterSpacing: -1 }}>Theresa</div>
          <div style={{ fontSize: 30, marginTop: 16, color: "rgba(255,255,255,0.85)", maxWidth: 480 }}>
            The AI tutor that teaches on a live board
          </div>
        </div>
      </div>
    ),
    { ...size }
  );
}
