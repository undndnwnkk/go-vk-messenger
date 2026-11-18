import type { ReactNode } from "react";

interface StatePanelProps {
  visual: ReactNode;
  title: string;
  description: string;
  children?: ReactNode;
}

export function StatePanel({ visual, title, description, children }: StatePanelProps) {
  return (
    <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-2 px-8 py-6 text-center">
      {visual}
      <h3 className="mt-3 font-heading text-[18px] leading-[26px] font-semibold">{title}</h3>
      <p className="m-0 max-w-[260px] text-base text-pretty text-text-2">{description}</p>
      {children && <div className="mt-3 flex w-full max-w-[220px] flex-col gap-2">{children}</div>}
    </div>
  );
}
