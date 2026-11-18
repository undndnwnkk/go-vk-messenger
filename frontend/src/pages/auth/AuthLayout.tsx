import type { ReactNode } from "react";
import { NavLink } from "react-router";

import { Logo } from "../../components/Logo";

function tabClass({ isActive }: { isActive: boolean }): string {
  const base = "flex h-9 flex-1 items-center justify-center rounded-[9px] text-base";
  return isActive
    ? `${base} bg-surface font-semibold text-text shadow-1`
    : `${base} font-medium text-text-2 hover:text-text`;
}

function Decoration() {
  return (
    <div className="relative hidden w-[42%] flex-none overflow-hidden bg-accent-soft desk:block">
      <svg
        width="100%"
        height="100%"
        viewBox="0 0 560 800"
        preserveAspectRatio="xMidYMid slice"
        aria-hidden="true"
        className="absolute inset-0 block"
      >
        <rect x="70" y="150" width="300" height="88" rx="26" fill="var(--surface)" />
        <rect x="96" y="176" width="200" height="10" rx="5" fill="var(--border)" />
        <rect x="96" y="198" width="130" height="10" rx="5" fill="var(--border)" />
        <rect x="190" y="262" width="300" height="66" rx="26" fill="var(--accent)" />
        <rect
          x="216"
          y="288"
          width="190"
          height="10"
          rx="5"
          fill="var(--accent-fg)"
          opacity="0.85"
        />
        <rect x="70" y="352" width="360" height="112" rx="26" fill="var(--surface)" />
        <rect x="96" y="378" width="270" height="10" rx="5" fill="var(--border)" />
        <rect x="96" y="400" width="300" height="10" rx="5" fill="var(--border)" />
        <rect x="96" y="422" width="160" height="10" rx="5" fill="var(--border)" />
        <rect
          x="330"
          y="446"
          width="76"
          height="32"
          rx="16"
          fill="var(--bg-app)"
          stroke="var(--border)"
        />
        <circle cx="352" cy="462" r="7" fill="var(--accent)" />
        <rect x="366" y="458" width="28" height="8" rx="4" fill="var(--text-3)" opacity=".5" />
        <rect x="250" y="502" width="250" height="66" rx="26" fill="var(--accent)" />
        <rect
          x="276"
          y="528"
          width="150"
          height="10"
          rx="5"
          fill="var(--accent-fg)"
          opacity="0.85"
        />
        <rect x="70" y="592" width="230" height="66" rx="26" fill="var(--surface)" />
        <rect x="96" y="618" width="150" height="10" rx="5" fill="var(--border)" />
      </svg>
    </div>
  );
}

interface AuthLayoutProps {
  title: string;
  subtitle: string;
  children: ReactNode;
  footer: ReactNode;
}

export function AuthLayout({ title, subtitle, children, footer }: AuthLayoutProps) {
  return (
    <div className="flex h-full overflow-hidden bg-surface">
      <div className="flex min-w-0 flex-1 flex-col overflow-y-auto px-6 py-6 desk:px-16 desk:py-8">
        <div className="flex-none">
          <Logo />
        </div>
        <main className="flex min-h-0 flex-1 items-center justify-center py-8">
          <div className="flex w-full max-w-[400px] flex-col gap-6">
            <nav aria-label="Account" className="flex gap-1 rounded-md bg-sunken p-1">
              <NavLink to="/login" className={tabClass}>
                Sign in
              </NavLink>
              <NavLink to="/register" className={tabClass}>
                Create account
              </NavLink>
            </nav>
            <div className="flex flex-col gap-1.5">
              <h1 className="m-0 font-heading text-2xl font-semibold tracking-[-0.01em]">
                {title}
              </h1>
              <p className="m-0 text-md text-text-2">{subtitle}</p>
            </div>
            {children}
            {footer}
          </div>
        </main>
        <div className="flex flex-none gap-5 text-sm text-text-3">
          <span>Plover</span>
        </div>
      </div>
      <Decoration />
    </div>
  );
}
