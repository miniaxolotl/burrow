import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { ImageResponse } from "next/og";

export const alt =
  "Burrow by Bramble Labs — Localhost, shared. A public HTTPS link for your local app.";
export const size = { width: 1200, height: 630 };
export const contentType = "image/png";

export default async function OpenGraphImage() {
  const [sans, bold, serif] = await Promise.all([
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
    readFile(
      join(
        process.cwd(),
        "node_modules/@fontsource/instrument-serif/files/instrument-serif-latin-400-italic.woff",
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
        background: "#faf7f2",
        color: "#352b28",
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
        <span style={{ fontSize: 18 }}>Bramble Labs</span>
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
          <span style={{ fontSize: 87, letterSpacing: -4, lineHeight: 1.1 }}>
            Localhost,
          </span>
          <span
            style={{
              fontFamily: "Instrument Serif",
              fontStyle: "italic",
              fontSize: 112,
              lineHeight: 1.1,
              color: "#9d4930",
            }}
          >
            shared.
          </span>
          <span style={{ marginTop: 26, fontSize: 22, color: "#74665e" }}>
            A public HTTPS link for your local app.
          </span>
        </div>
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            width: 260,
            height: 260,
            borderRadius: 12,
            background: "#f1dace",
          }}
        >
          <svg width="170" height="170" viewBox="0 0 40 40" fill="none">
            <path
              d="M5 33V20a15 15 0 0 1 30 0v13M12 33V20a8 8 0 0 1 16 0v13M20 33V21"
              stroke="#9d4930"
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
          borderTop: "1px solid #e5dbd1",
          paddingTop: 24,
          fontSize: 13,
          letterSpacing: 2,
          color: "#74665e",
        }}
      >
        <span>HTTPS TUNNELS FOR LOCAL DEVELOPMENT</span>
        <span>A BRAMBLE LABS APP</span>
      </div>
    </div>,
    {
      ...size,
      fonts: [
        { name: "DM Sans", data: sans, weight: 400, style: "normal" },
        { name: "DM Sans", data: bold, weight: 700, style: "normal" },
        { name: "Instrument Serif", data: serif, weight: 400, style: "italic" },
      ],
    },
  );
}
