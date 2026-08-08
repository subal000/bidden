import type { Config } from "tailwindcss";
export default {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}"],
  theme: {
    extend: {
      fontFamily: { mono: ["ui-monospace", "SFMono-Regular", "Menlo", "monospace"] },
      colors: {
        ink: "#07090d",
        raise: "#141a24",
        panel: "#0e1219",
        edge: "#1c2431",
        dim: "#5d6b7f",
        pale: "#9fb0c6",
        accent: "#4ade80",
        warn: "#fbbf24",
        hot: "#f472b6",
      },
    },
  },
  plugins: [],
} satisfies Config;
