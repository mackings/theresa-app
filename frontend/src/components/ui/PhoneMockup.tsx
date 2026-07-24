export function PhoneMockup({ children }: { children: React.ReactNode }) {
  return (
    <div className="relative mx-auto w-[260px] rounded-[2.5rem] border-[8px] border-[#111214] bg-[#111214] p-2 shadow-[var(--shadow-lg)]">
      <div className="relative overflow-hidden rounded-[1.75rem] bg-[var(--color-bg)]">
        <div className="absolute left-1/2 top-2 z-10 h-5 w-20 -translate-x-1/2 rounded-full bg-[#111214]" />
        <div className="flex items-center justify-between px-5 pb-1 pt-3 text-[11px] font-medium text-[var(--color-text-primary)]">
          <span>9:41</span>
          <span className="opacity-60">●●●●</span>
        </div>
        {children}
      </div>
    </div>
  );
}
