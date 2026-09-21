import type { Config } from "tailwindcss";

// Design tokens for the AI Resume Parser & Intelligent Recruitment
// Platform. Palette rationale: navy + warm amber reads as "credential /
// letterhead" rather than the generic SaaS teal-gradient or purple-AI
// look, while still honoring the brief's "blue/white primary" direction.
// Amber is used sparingly (primary CTA, active states, score highlights)
// so it stays a genuine accent rather than competing with navy.
const config: Config = {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        navy: {
          950: "#0B1526",
          900: "#10233F",
          800: "#173257",
          700: "#204373",
          600: "#2F5FA3",
          500: "#4676C2",
          300: "#8FADD6",
          200: "#C3D4EB",
          50: "#EEF3FA",
        },
        amber: {
          700: "#A66C15",
          600: "#C9861F",
          500: "#F2A93B",
          400: "#F6C266",
          100: "#FDF0DA",
        },
        ink: "#121826",
        paper: "#F7F8FA",
        line: "#E4E7EC",
      },
      fontFamily: {
        sans: [
          "-apple-system",
          "BlinkMacSystemFont",
          "Segoe UI",
          "Inter",
          "Roboto",
          "Helvetica Neue",
          "Arial",
          "sans-serif",
        ],
      },
      fontSize: {
        "display-lg": ["4.25rem", { lineHeight: "1.02", letterSpacing: "-0.02em", fontWeight: "800" }],
        "display-md": ["3rem", { lineHeight: "1.05", letterSpacing: "-0.015em", fontWeight: "800" }],
        "display-sm": ["2rem", { lineHeight: "1.1", letterSpacing: "-0.01em", fontWeight: "700" }],
      },
      boxShadow: {
        card: "0 1px 2px rgba(16, 35, 63, 0.06), 0 1px 1px rgba(16, 35, 63, 0.04)",
        raised: "0 8px 24px rgba(16, 35, 63, 0.12)",
      },
      borderRadius: {
        card: "10px",
      },
    },
  },
  plugins: [],
};

export default config;
