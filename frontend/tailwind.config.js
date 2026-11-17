/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,jsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      // ── Brand palette ───────────────────────────────────────────────────
      // Playful & colorful: indigo primary + sakura pink + matcha green
      // accents, each with 50–900 steps for flexible UI.
      colors: {
        indigo: { 50: '#fff0f1', 100: '#ffdde0', 200: '#ffc5cc', 300: '#ff8494', 400: '#ed425c', 500: '#e51c3a', 600: '#c8102e', 700: '#a50923', 800: '#800d23', 900: '#580d1c' },
        violet: { 50: '#fff0f1', 100: '#ffdde0', 200: '#ffc5cc', 300: '#ff8494', 400: '#ed425c', 500: '#e51c3a', 600: '#b70c29', 700: '#93031d', 800: '#800d23', 900: '#580d1c' },
        sakura: {
          50: '#FFF1F5',
          100: '#FFE4EC',
          200: '#FECDD9',
          300: '#FDA4BD',
          400: '#FB6F9A',
          500: '#F43F75',
          600: '#E11D57',
          700: '#BE1247',
          800: '#9F1240',
          900: '#88143D',
        },
        matcha: {
          50: '#F3FBE9',
          100: '#E2F7C9',
          200: '#CFEF9F',
          300: '#B5E26B',
          400: '#97CE42',
          500: '#7BB327',
          600: '#5C911B',
          700: '#477017',
          800: '#3A5817',
          900: '#314B18',
        },
        sun: {
          50: '#FFFBEB',
          100: '#FFF3C4',
          200: '#FCE588',
          300: '#FBD24E',
          400: '#FABF25',
          500: '#E9A20C',
          600: '#CC7F04',
          700: '#A15607',
          800: '#86400C',
          900: '#73370D',
        },
      },
      // ── Typography ──────────────────────────────────────────────────────
      fontFamily: {
        jp: ['"Noto Sans JP"', 'sans-serif'],
      },
      // ── Shadows (soft, layered — playful feel) ──────────────────────────
      boxShadow: {
        soft: '0 2px 8px -1px rgb(0 0 0 / 0.04), 0 1px 3px -1px rgb(0 0 0 / 0.06)',
        'soft-md': '0 4px 16px -2px rgb(0 0 0 / 0.06), 0 2px 6px -2px rgb(0 0 0 / 0.08)',
        'soft-lg': '0 8px 32px -4px rgb(0 0 0 / 0.08), 0 4px 12px -4px rgb(0 0 0 / 0.1)',
        glow: '0 0 0 3px rgb(99 102 241 / 0.12)',
        'glow-sakura': '0 0 0 3px rgb(244 63 117 / 0.12)',
      },
      // ── Border radius (generous, rounded = playful) ────────────────────
      borderRadius: {
        '2.5xl': '1.25rem',
        '3.5xl': '1.75rem',
      },
      // ── Animations ──────────────────────────────────────────────────────
      keyframes: {
        'pop-in': {
          '0%': { transform: 'scale(0.95)', opacity: '0' },
          '100%': { transform: 'scale(1)', opacity: '1' },
        },
        'wiggle': {
          '0%, 100%': { transform: 'rotate(0deg)' },
          '25%': { transform: 'rotate(-2deg)' },
          '75%': { transform: 'rotate(2deg)' },
        },
      },
      animation: {
        'pop-in': 'pop-in 0.25s ease-out',
        'wiggle': 'wiggle 0.3s ease-in-out',
      },
    },
  },
  plugins: [],
};