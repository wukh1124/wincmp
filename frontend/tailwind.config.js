/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        darkBg: '#09090B',
        darkCard: '#18181B',
        darkInput: '#121214',
        darkBorder: '#27272A',
      }
    },
  },
  plugins: [],
}
