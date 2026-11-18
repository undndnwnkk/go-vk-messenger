interface LogoProps {
  size?: "md" | "lg";
}

export function Logo({ size = "lg" }: LogoProps) {
  return (
    <div className="flex items-center gap-2.5">
      <svg
        width="32"
        height="32"
        viewBox="0 0 32 32"
        aria-hidden="true"
        className="block flex-none"
      >
        <path
          d="M16 3C8.8 3 3 8.3 3 15c0 3.2 1.4 6.1 3.7 8.2L5.5 28.5l5.4-2.4c1.6.5 3.3.8 5.1.8 7.2 0 13-5.3 13-12S23.2 3 16 3z"
          fill="var(--accent)"
        />
        <circle cx="16" cy="15" r="3.4" fill="var(--accent-fg)" />
      </svg>
      <span
        className={`font-heading leading-none font-semibold tracking-[-0.02em] text-text ${size === "lg" ? "text-[24px]" : "text-[22px]"}`}
      >
        plover
      </span>
    </div>
  );
}
