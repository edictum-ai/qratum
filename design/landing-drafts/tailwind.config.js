/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      fontFamily: {
        sans: ["Inter", "ui-sans-serif", "system-ui", "sans-serif"],
        mono: ["'JetBrains Mono'", "ui-monospace", "monospace"],
        serif: ["Fraunces", "Lora", "ui-serif", "Georgia", "serif"],
        display: ["'Space Grotesk'", "Inter", "sans-serif"],
        reading: ["Lora", "Georgia", "serif"],
      },
    },
  },
  plugins: [],
};
