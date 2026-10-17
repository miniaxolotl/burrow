import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { ImageResponse } from "next/og";

export const alt =
  "Burrow by Bramble Labs Inc. — Localhost, shared. A public HTTPS link for your local app.";
export const size = { width: 1200, height: 630 };
export const contentType = "image/png";

export default async function OpenGraphImage() {
  const [sans, bold] = await Promise.all([
    readFile(
      join(
        process.cwd(),
        "node_modules/@fontsource/dm-sans/files/dm-sans-latin-400-normal.woff",
      ),
    ),
    readFile(
      join(
        process.cwd(),
        "node_modules/@fontsource/dm-sans/files/dm-sans-latin-700-normal.woff",
      ),
    ),
  ]);
  return new ImageResponse(
    <div
      style={{
        width: "100%",
        height: "100%",
        display: "flex",
        flexDirection: "column",
        background: "#f7f8f5",
        color: "#1d2d26",
        padding: 60,
        fontFamily: "DM Sans",
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
        }}
      >
        <span style={{ fontSize: 38, fontWeight: 700, letterSpacing: -2 }}>
          burrow
        </span>
        <span style={{ fontSize: 18 }}>Bramble Labs Inc.</span>
      </div>
      <div
        style={{
          display: "flex",
          flex: 1,
          alignItems: "center",
          justifyContent: "space-between",
        }}
      >
        <div style={{ display: "flex", flexDirection: "column", width: 750 }}>
          <span
            style={{
              fontSize: 88,
              fontWeight: 700,
              letterSpacing: -2,
              lineHeight: 1.04,
            }}
          >
            Localhost,
          </span>
          <span
            style={{
              fontWeight: 700,
              fontSize: 88,
              letterSpacing: -2,
              lineHeight: 1.04,
              color: "#9d4930",
            }}
          >
            shared.
          </span>
          <span style={{ marginTop: 26, fontSize: 22, color: "#5d6962" }}>
            A public HTTPS link for your local app.
          </span>
        </div>
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            width: 200,
            height: 200,
            borderRadius: 12,
            background: "#1d2d26",
          }}
        >
          <svg width="120" height="120" viewBox="0 0 40 40" fill="none">
            <path
              d="M5 33V20a15 15 0 0 1 30 0v13M12 33V20a8 8 0 0 1 16 0v13M20 33V21"
              stroke="#efbc96"
              strokeWidth="2"
              strokeLinecap="round"
            />
          </svg>
        </div>
      </div>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          borderTop: "1px solid #dce2db",
          paddingTop: 24,
          fontSize: 13,
          letterSpacing: 2,
          color: "#5d6962",
        }}
      >
        <span>HTTPS TUNNELS FOR LOCAL DEVELOPMENT</span>
        <span>DEVELOPER TOOLS</span>
      </div>
    </div>,
    {
      ...size,
      fonts: [
        { name: "DM Sans", data: sans, weight: 400, style: "normal" },
        { name: "DM Sans", data: bold, weight: 700, style: "normal" },
      ],
    },
  );
}
