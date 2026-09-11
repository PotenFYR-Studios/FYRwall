/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  darkMode: "class",
  theme: {
    extend: {
      colors: {
        fyr: {
          50: "#eef7ff",
          500: "#f97316",
          600: "#ea580c",
          900: "#7c2d12",
        },
      },
    },
  },
  plugins: [],
};
