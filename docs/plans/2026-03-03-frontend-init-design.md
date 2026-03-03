# Frontend Init Design

## Goal

Initialize `frontend-real/` as the production frontend for MyEnglish. Clean scaffold, no pages or business components — infrastructure only.

## Stack

- Vite + React 18 + TypeScript
- Tailwind CSS v3 with custom token config
- shadcn/ui (copy-paste model)
- React Router DOM v6
- Lucide React
- Framer Motion
- cmdk
- canvas-confetti

## Steps

1. Scaffold with `npm create vite@latest frontend-real -- --template react-ts`
2. Install all dependencies
3. Init shadcn/ui
4. Configure `tailwind.config.ts` with Herbarium color tokens
5. Set up CSS variables in `src/index.css`
6. Import Google Fonts (Space Grotesk, Orelega One, EB Garamond, Courier Prime)
7. Create file structure per design system
8. Add base router and empty App

## File Structure

```
src/
  components/
    ui/       # shadcn
    common/
  pages/
  hooks/
  lib/
  types/
  graphql/
  router/
  assets/
```

## Out of Scope

No pages, no business components. Registration page is a separate task.
