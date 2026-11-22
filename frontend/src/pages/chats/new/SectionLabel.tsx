export function SectionLabel({ children }: { children: string }) {
  return (
    <div className="px-6 pt-3 pb-1 text-xs font-semibold tracking-[0.04em] text-text-2 uppercase">
      {children}
    </div>
  );
}
