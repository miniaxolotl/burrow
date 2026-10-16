import Link from "next/link";

export function BurrowMark({ className = "" }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 40 40"
      fill="none"
      className={className}
      aria-hidden="true"
    >
      <path
        d="M5 33V20a15 15 0 0 1 30 0v13M12 33V20a8 8 0 0 1 16 0v13M20 33V21"
        stroke="currentColor"
        strokeWidth="3.5"
        strokeLinecap="round"
      />
    </svg>
  );
}

export function Brand({ light = false }: { light?: boolean }) {
  return (
    <Link
      href="/"
      aria-label="Burrow home"
      className={`inline-flex shrink-0 items-center gap-2 transition-opacity hover:opacity-70 sm:gap-2.5 ${light ? "text-paper" : "text-forest"}`}
    >
      <BurrowMark className="size-7 sm:size-9" />
      <span className="text-[25px] font-bold tracking-[-1.5px] sm:text-[29px]">
        burrow
      </span>
    </Link>
  );
}
