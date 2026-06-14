import type { Variants } from "framer-motion";

// Shared easing + variants. Drafts may use these or define their own, but the
// ease curve keeps motion feeling consistent across the gallery.
export const easeOut = [0.22, 1, 0.36, 1] as const;

export const fadeUp: Variants = {
  hidden: { opacity: 0, y: 18 },
  show: { opacity: 1, y: 0, transition: { duration: 0.55, ease: easeOut } },
};

export const fadeIn: Variants = {
  hidden: { opacity: 0 },
  show: { opacity: 1, transition: { duration: 0.6, ease: easeOut } },
};

export const stagger: Variants = {
  hidden: {},
  show: { transition: { staggerChildren: 0.08, delayChildren: 0.05 } },
};

// Viewport config for whileInView reveals.
export const viewport = { once: true, amount: 0.3 } as const;
