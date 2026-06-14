import { useEffect, useRef, useState } from "react";

/**
 * Dark Ledger — count-up, adapted from the vault draft. Eases toward the
 * target when scrolled into view; handles non-numeric suffixes ("30d" → 30 + "d").
 * Reduced-motion shows the final value immediately.
 */
export default function CountUp({
  value,
  className,
  duration = 1500,
}: {
  value: string;
  className?: string;
  duration?: number;
}) {
  const match = value.match(/^(\d+)(.*)$/);
  const num = match ? parseInt(match[1], 10) : 0;
  const suffix = match ? match[2] : "";
  const isNumeric = match !== null;

  const [display, setDisplay] = useState<number>(() => 0);
  const [started, setStarted] = useState(false);
  const ref = useRef<HTMLSpanElement>(null);

  const reduced =
    typeof window !== "undefined" &&
    window.matchMedia("(prefers-reduced-motion: reduce)").matches;

  useEffect(() => {
    if (reduced || !isNumeric) {
      setDisplay(num);
      return;
    }
    const el = ref.current;
    if (!el) return;
    const io = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting) {
          setStarted(true);
          io.disconnect();
        }
      },
      { threshold: 0.4 }
    );
    io.observe(el);
    return () => io.disconnect();
  }, [num, isNumeric, reduced]);

  useEffect(() => {
    if (!started || reduced) return;
    let raf = 0;
    const start = performance.now();
    const tick = (now: number) => {
      const p = Math.min(1, (now - start) / duration);
      // easeOutCubic — calm on dark.
      const eased = 1 - Math.pow(1 - p, 3);
      setDisplay(Math.round(num * eased));
      if (p < 1) raf = requestAnimationFrame(tick);
    };
    raf = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(raf);
  }, [started, num, duration, reduced]);

  return (
    <span ref={ref} className={className}>
      {isNumeric ? (
        <>
          {display.toLocaleString()}
          {suffix}
        </>
      ) : (
        value
      )}
    </span>
  );
}
