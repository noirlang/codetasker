import type { Config } from 'tailwindcss';

const config: Config = {
  // Paths that Tailwind should scan for class names
  content: [
    './index.html',
    './src/**/*.{ts,tsx}',
  ],

  theme: {
    extend: {
      // ─── Color palette ───────────────────────────────────────────────────
      colors: {
        background: {
          DEFAULT: '#0a0a0a', // page background
          panel:   '#111111', // side panels / nav
          card:    '#1a1a1a', // cards / list items
          elevated:'#242424', // elevated cards / selected states
        },
        border: {
          subtle:  '#2a2a2a', // barely visible dividers
          DEFAULT: '#3a3a3a', // standard visible border
          strong:  '#4a4a4a', // emphasized border on hover/focus
        },
        text: {
          primary:   '#ffffff', // headings, labels
          secondary: '#a0a0a0', // supporting text
          muted:     '#666666', // timestamps, metadata
        },
      },

      // ─── Typography ──────────────────────────────────────────────────────
      fontFamily: {
        sans: ['Inter', 'sans-serif'],
        mono: ['JetBrains Mono', 'Menlo', 'Monaco', 'Consolas', 'monospace'],
      },

      // ─── Animations ──────────────────────────────────────────────────────
      animation: {
        'fade-in':  'fadeIn 150ms ease',
        'slide-in': 'slideIn 200ms ease',
      },
      keyframes: {
        fadeIn: {
          from: { opacity: '0' },
          to:   { opacity: '1' },
        },
        slideIn: {
          from: { transform: 'translateX(100%)' },
          to:   { transform: 'translateX(0)' },
        },
      },

      // ─── Transitions ─────────────────────────────────────────────────────
      transitionDuration: {
        DEFAULT: '150ms',
      },
    },
  },

  plugins: [],
};

export default config;
