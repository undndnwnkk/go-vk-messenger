import type { ReactNode } from "react";

const icons = {
  compose: (
    <>
      <path d="M11 4H7a3 3 0 00-3 3v10a3 3 0 003 3h10a3 3 0 003-3v-4" />
      <path d="M17.2 3.8a1.9 1.9 0 012.7 2.7L12 14.4 8.5 15.5l1.1-3.5 7.6-8.2z" />
    </>
  ),
  moon: <path d="M20 14.5A8.5 8.5 0 019.5 4 8.5 8.5 0 1020 14.5z" />,
  sun: (
    <>
      <circle cx="12" cy="12" r="4" />
      <path d="M12 3v2M12 19v2M3 12h2M19 12h2M5.6 5.6L7 7M17 17l1.4 1.4M18.4 5.6L17 7M7 17l-1.4 1.4" />
    </>
  ),
  signOut: (
    <>
      <path d="M10 5H6.5A2.5 2.5 0 004 7.5v9A2.5 2.5 0 006.5 19H10" />
      <path d="M15 8l4 4-4 4M19 12H9.5" />
    </>
  ),
  search: (
    <>
      <circle cx="11" cy="11" r="6.5" />
      <path d="M16 16l4.5 4.5" />
    </>
  ),
  group: (
    <>
      <circle cx="9" cy="9" r="3.3" />
      <path d="M3 19.5c.6-3 3-4.8 6-4.8s5.4 1.8 6 4.8" />
      <circle cx="17" cy="9.5" r="2.6" />
      <path d="M16.5 14.8c2.4.2 4.1 1.7 4.5 4" />
    </>
  ),
  back: <path d="M14.5 6L8.5 12l6 6" />,
  close: <path d="M6 6l12 12M18 6L6 18" />,
  eye: (
    <>
      <path d="M2.5 12S6 5.5 12 5.5 21.5 12 21.5 12 18 18.5 12 18.5 2.5 12 2.5 12z" />
      <circle cx="12" cy="12" r="2.8" />
    </>
  ),
  eyeOff: (
    <>
      <path d="M2.5 12S6 5.5 12 5.5 21.5 12 21.5 12 18 18.5 12 18.5 2.5 12 2.5 12z" />
      <circle cx="12" cy="12" r="2.8" />
      <path d="M4 4l16 16" />
    </>
  ),
  alert: (
    <>
      <circle cx="12" cy="12" r="8.5" />
      <path d="M12 7.8v5" />
      <path d="M12 16.2h.01" strokeWidth={2.4} />
    </>
  ),
  retry: (
    <>
      <path d="M20 11a8 8 0 10-2.3 5.7" />
      <path d="M20 5v6h-6" />
    </>
  ),
  muted: (
    <>
      <path d="M6 16.5V11a6 6 0 0112 0v5.5l1.5 2h-15l1.5-2z" />
      <path d="M10 21h4" />
      <path d="M4 4l16 16" />
    </>
  ),
} satisfies Record<string, ReactNode>;

export type IconName = keyof typeof icons;

interface IconProps {
  name: IconName;
  size?: number;
  strokeWidth?: number;
  className?: string;
}

export function Icon({ name, size = 20, strokeWidth = 1.7, className }: IconProps) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={strokeWidth}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      className={["block flex-none", className].filter(Boolean).join(" ")}
    >
      {icons[name]}
    </svg>
  );
}
