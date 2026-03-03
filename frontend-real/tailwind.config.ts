import type { Config } from 'tailwindcss'

export default {
  darkMode: ['class'],
  content: [
    './index.html',
    './src/**/*.{ts,tsx}',
  ],
  theme: {
    extend: {
      colors: {
        'bg-page': 'var(--bg-page)',
        'bg-card': 'var(--bg-card)',
        'surface-secondary': 'var(--surface-secondary)',
        'surface-disabled': 'var(--surface-disabled)',
        'border-default': 'var(--border-default)',
        'border-subtle': 'var(--border-subtle)',
        'text-primary': 'var(--text-primary)',
        'text-secondary': 'var(--text-secondary)',
        'text-tertiary': 'var(--text-tertiary)',
        'text-disabled': 'var(--text-disabled)',
        'text-on-accent': 'var(--text-on-accent)',
        poppy: {
          DEFAULT: 'var(--poppy)',
          hover: 'var(--poppy-hover)',
          light: 'var(--poppy-light)',
          fg: 'var(--poppy-fg)',
        },
        goldenrod: {
          DEFAULT: 'var(--goldenrod)',
          hover: 'var(--goldenrod-hover)',
          light: 'var(--goldenrod-light)',
          fg: 'var(--goldenrod-fg)',
        },
        cornflower: {
          DEFAULT: 'var(--cornflower)',
          hover: 'var(--cornflower-hover)',
          light: 'var(--cornflower-light)',
          fg: 'var(--cornflower-fg)',
        },
        thyme: {
          DEFAULT: 'var(--thyme)',
          hover: 'var(--thyme-hover)',
          light: 'var(--thyme-light)',
          fg: 'var(--thyme-fg)',
        },
      },
      fontFamily: {
        sans: ['Space Grotesk', 'system-ui', 'sans-serif'],
        serif: ['EB Garamond', 'Georgia', 'serif'],
        mono: ['Courier Prime', 'ui-monospace', 'monospace'],
        orelega: ['Orelega One', 'Georgia', 'serif'],
      },
    },
  },
  plugins: [require('tailwindcss-animate')],
} satisfies Config
