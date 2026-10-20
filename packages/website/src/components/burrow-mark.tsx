import artwork from "@/lib/brand-artwork.json";

export function BurrowMark({
  className = "",
  size = 36,
  color = "currentColor",
}: {
  className?: string;
  size?: number;
  color?: string;
}) {
  return (
    <svg
      width={size}
      height={size}
      viewBox={artwork.mark.viewBox}
      className={className}
      aria-hidden="true"
      focusable="false"
    >
      <g fill={color} fillRule="evenodd" transform={artwork.mark.transform}>
        {artwork.mark.paths.map((path, index) => (
          <path key={index} d={path} />
        ))}
      </g>
    </svg>
  );
}
