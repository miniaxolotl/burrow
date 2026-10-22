import Link from "next/link";
import { BurrowLogo } from "@/components/burrow-logo";
import artwork from "@/lib/brand-artwork.json";

export { BurrowMark } from "@/components/burrow-mark";

export function Brand({ light = false }: { light?: boolean }) {
  return (
    <Link
      href="/"
      aria-label="Burrow home"
      className={`site-brand inline-flex min-h-11 shrink-0 items-center transition-opacity hover:opacity-70 ${light ? "text-paper" : "text-forest"}`}
    >
      <BurrowLogo
        className="h-6 w-auto min-[360px]:h-7 sm:h-9"
        color={light ? artwork.palette.paper : artwork.palette.forest}
        accent={light ? artwork.palette.paper : artwork.palette.clay}
      />
    </Link>
  );
}
