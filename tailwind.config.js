/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./templates/**/*.html", // HTMX templates
    "./**/*.go", // optional JS files
  ],
  theme: {
    extend: {},
  },
  plugins: [],
};
