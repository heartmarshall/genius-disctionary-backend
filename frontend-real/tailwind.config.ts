import type { Config } from 'tailwindcss'
import tailwindcssAnimate from 'tailwindcss-animate'

export default {
  darkMode: ['class'],
  content: [
    './index.html',
    './src/**/*.{ts,tsx}',
  ],
  theme: {
    extend: {
      colors: {
        /* shadcn semantic tokens */
        background: 'var(--background)',
        foreground: 'var(--foreground)',
        card: { DEFAULT: 'var(--card)', foreground: 'var(--card-foreground)' },
        popover: { DEFAULT: 'var(--popover)', foreground: 'var(--popover-foreground)' },
        primary: { DEFAULT: 'var(--primary)', foreground: 'var(--primary-foreground)' },
        secondary: { DEFAULT: 'var(--secondary)', foreground: 'var(--secondary-foreground)' },
        muted: { DEFAULT: 'var(--muted)', foreground: 'var(--muted-foreground)' },
        accent: { DEFAULT: 'var(--accent)', foreground: 'var(--accent-foreground)' },
        destructive: { DEFAULT: 'var(--destructive)', foreground: 'var(--destructive-foreground)' },
        border: 'var(--border)',
        input: 'var(--input)',
        ring: 'var(--ring)',

        /* Herbarium Layer 1 — Interface */
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
        'source-book': { DEFAULT: 'var(--source-book)', light: 'var(--source-book-light)' },
        'source-screen': { DEFAULT: 'var(--source-screen)', light: 'var(--source-screen-light)' },
        'source-music': { DEFAULT: 'var(--source-music)', light: 'var(--source-music-light)' },
      },
      fontFamily: {
        sans: ['Space Grotesk', 'system-ui', 'sans-serif'],
        serif: ['EB Garamond', 'Georgia', 'serif'],
        mono: ['Courier Prime', 'ui-monospace', 'monospace'],
        orelega: ['Orelega One', 'Georgia', 'serif'],
      },
    },
  },
  plugins: [tailwindcssAnimate],
} satisfies Config
