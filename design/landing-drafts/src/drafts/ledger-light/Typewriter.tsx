import { useEffect, useState } from "react";

/**
 * Dark Ledger — typewriter reveal for the hero subtitle, adapted from the
 * vault draft. Reduced-motion returns the full string immediately.
 */
export default function Typewriter({
  text,
  speed = 24,
  className,
  showCaret = true,
  onDone,
}: {
  text: string;
  speed?: number;
  className?: string;
  showCaret?: boolean;
  onDone?: () => void;
}) {
  const reduced =
    typeof window !== "undefined" &&
    window.matchMedia("(prefers-reduced-motion: reduce)").matches;

  const [count, setCount] = useState<number>(() => (reduced ? text.length : 0));

  useEffect(() => {
    if (reduced) {
      onDone?.();
      return;
    }
    let i = 0;
    const id = setInterval(() => {
      i += 1;
      setCount(i);
      if (i >= text.length) {
        clearInterval(id);
        onDone?.();
      }
    }, speed);
    return () => clearInterval(id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [text, speed, reduced]);

  const done = count >= text.length;

  return (
    <span className={className}>
      {text.slice(0, count)}
      {showCaret && !done && <span className="dl-caret" aria-hidden="true" />}
    </span>
  );
}
