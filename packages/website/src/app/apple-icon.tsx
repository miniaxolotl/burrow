import { ImageResponse } from "next/og";

export const size = { width: 180, height: 180 };
export const contentType = "image/png";

export default function AppleIcon() {
  return new ImageResponse(
    <div
      style={{
        display: "flex",
        width: "100%",
        height: "100%",
        background: "#352b28",
        alignItems: "center",
        justifyContent: "center",
      }}
    >
      <svg width="116" height="116" viewBox="0 0 40 40" fill="none">
        <path
          d="M5 33V20a15 15 0 0 1 30 0v13M12 33V20a8 8 0 0 1 16 0v13M20 33V21"
          stroke="#efbc96"
          strokeWidth="3.5"
          strokeLinecap="round"
        />
      </svg>
    </div>,
    size,
  );
}
