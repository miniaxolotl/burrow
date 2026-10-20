import artwork from "@/lib/brand-artwork.json";

// One SVG preserves the approved symbol spacing and outlined wordmark at every size.
export function BurrowLogo({
  className = "",
  height = 36,
  color = "currentColor",
  accent = color,
}: {
  className?: string;
  height?: number;
  color?: string;
  accent?: string;
}) {
  return (
    <svg
      width={(height * artwork.logo.width) / artwork.logo.height}
      height={height}
      viewBox={artwork.logo.viewBox}
      className={className}
      aria-hidden="true"
      focusable="false"
    >
      <g
        fill={accent}
        fillRule="evenodd"
        transform={artwork.logo.markTransform}
      >
        {artwork.mark.paths.map((path, index) => (
          <path key={index} d={path} />
        ))}
      </g>
      <path
        fill={color}
        transform={artwork.logo.wordTransform}
        d={artwork.wordmark.path}
      />
    </svg>
  );
}
