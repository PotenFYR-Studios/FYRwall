/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  darkMode: "class",
  theme: {
    extend: {
      colors: {
        fyr: {
          50: "#eef7ff",
          100: "#dcf2ff",
          200: "#b8e4ff",
          300: "#7acdfd",
          400: "#38adf6",
          500: "#f97316",
          600: "#ea580c",
          700: "#c2410c",
          800: "#9a3412",
          900: "#7c2d12",
        },
      },
      animation: {
        "spin-slow": "spin 8s linear infinite",
        "pulse-slow": "pulse 4s cubic-bezier(0.4, 0, 0.6, 1) infinite",
        shimmer: "shimmer 2s linear infinite",
        "border-beam": "border-beam calc(var(--duration)*1s) infinite linear",
        orbit: "orbit calc(var(--duration)*1s) linear infinite",
        marquee: "marquee var(--duration) linear infinite",
        gradient: "gradient 8s linear infinite",
      },
      keyframes: {
        shimmer: {
          from: { backgroundPosition: "0 0" },
          to: { backgroundPosition: "-200% 0" },
        },
        "border-beam": {
          "100%": { "offset-distance": "100%" },
        },
        orbit: {
          "0%": {
            transform:
              "rotate(calc(var(--angle)*1deg)) translateY(calc(var(--radius)*1px)) rotate(calc(var(--angle)*-1deg))",
          },
          "100%": {
            transform:
              "rotate(calc(var(--angle)*1deg + 360deg)) translateY(calc(var(--radius)*1px)) rotate(calc((var(--angle)*-1deg) - 360deg))",
          },
        },
        marquee: {
          from: { transform: "translateX(0)" },
          to: { transform: "translateX(calc(-100% - var(--gap)))" },
        },
        gradient: {
          to: { backgroundPosition: "var(--bg-size) 0" },
        },
      },
    },
  },
  plugins: [require("tailwindcss-animate")],
};
