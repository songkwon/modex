import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}", "./lib/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        border: "hsl(var(--border))",
        background: "hsl(var(--background))",
        foreground: "hsl(var(--foreground))",
        muted: "hsl(var(--muted))",
        "muted-panel": "hsl(var(--muted-panel))",
        panel: "hsl(var(--panel))",
        "panel-subtle": "hsl(var(--panel-subtle))",
        accent: "hsl(var(--accent))"
      }
    }
  },
  plugins: []
};

export default config;
