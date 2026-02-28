# Phase 0: Foundation — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Runnable empty project with design system, routing, Apollo Client, auth infrastructure, and base UI components — ready for feature development in any phase.

**Architecture:** SPA with React Router (layout + page stubs), Apollo Client for GraphQL with JWT auth link and automatic token refresh, Tailwind CSS consuming Herbarium CSS variables for theming. shadcn/ui provides accessible component primitives, customized to match Herbarium design system.

**Tech Stack:** React 19, TypeScript, Vite, Tailwind CSS v4, shadcn/ui, Apollo Client, React Router v7

**Key references:**
- Design system: `frontend-real/design-docs/palette-v3.html` — all tokens, component examples
- API: `backend_v4/docs/API.md` — endpoints, auth flow
- Business rules: `docs/business/BUSINESS_RULES.md` — validation limits
- General plan: `docs/plans/2026-02-28-frontend-plan-design.md` — phase overview

---

## Task 1: Project Init

**Goal:** Scaffolded Vite + React + TypeScript project with clean structure.

**What to do:**
- Run `npm create vite@latest` inside `frontend-real/` with React + TypeScript template
- Clean up boilerplate (remove demo styles, App.tsx content, vite.svg, etc.)
- Create project directory structure:

```
frontend-real/
├── src/
│   ├── components/       # Reusable UI components
│   │   └── ui/           # shadcn/ui components (auto-generated)
│   ├── pages/            # Page components (one per route)
│   ├── layouts/          # Layout wrappers (MainLayout, AuthLayout)
│   ├── hooks/            # Custom React hooks
│   ├── lib/              # Utilities, helpers, constants
│   ├── graphql/          # GraphQL queries, mutations, fragments
│   ├── providers/        # Context providers (Auth, Apollo, Theme)
│   ├── types/            # Shared TypeScript types
│   ├── styles/           # Global CSS, Herbarium tokens
│   ├── App.tsx
│   └── main.tsx
├── public/
├── index.html
├── package.json
├── tsconfig.json
├── vite.config.ts
└── .gitignore
```

- Add `.gitignore` covering `node_modules/`, `dist/`, `.env`, `.env.local`
- Verify `npm run dev` starts successfully

**Commit:** `feat(frontend): scaffold Vite + React + TypeScript project`

---

## Task 2: Tailwind CSS + Herbarium Design Tokens

**Goal:** Tailwind configured with all Herbarium CSS variables available as utility classes.

**Context:** The design system in `palette-v3.html` defines ~60 CSS custom properties in `:root` — colors (oklch), spacing, radius, type scale, motion, elevation, layout, z-index. We keep them as CSS variables and reference them from Tailwind config so both systems coexist. Tailwind v4 uses CSS-based configuration (not `tailwind.config.js`).

**What to do:**
- Install Tailwind CSS v4 + `@tailwindcss/vite` plugin
- Create `src/styles/herbarium.css` — extract ALL CSS variables from `palette-v3.html` `:root` block (lines 10–163 of the HTML). This includes:
  - Flowers (poppy, goldenrod, cornflower, thyme + hover/light/fg variants)
  - Earth & Wood (umber, bark, clay + light variants)
  - Greens (moss, fern, sage + light)
  - Petals & Berry (dried-rose, lavender, heather + light)
  - Warm Neutrals (parchment, linen, honey, straw)
  - Cool Neutrals (mist, slate, storm)
  - Semantic aliases (bg-page, bg-card, bg-surface, text-primary/secondary/tertiary/disabled, border, accent, srs-*, status-*, danger/warning/success, source-*)
  - Spacing scale (2xs through 2xl)
  - Radius (sm, md, lg, full)
  - Type scale (2xl through xs + leading)
  - Motion (durations + easings)
  - Elevation (0-3)
  - Layout (container-max, container-narrow, sidebar-width, tab-bar-height)
  - Z-index layers
- Create `src/styles/globals.css` — import Tailwind + herbarium tokens, set base body styles (font-family: Space Grotesk, color: var(--text-primary), background: var(--bg-page))
- Configure `@theme` in Tailwind CSS to map Herbarium variables to Tailwind utilities. Key mappings:
  - Colors: `--color-poppy: var(--poppy)`, etc. — so `bg-poppy`, `text-umber` work
  - Font families: `--font-heading: 'Orelega One', serif`, `--font-body: 'Space Grotesk', sans-serif`, `--font-word: 'Lisu Bosa', serif`, `--font-example-book: 'EB Garamond', serif`, `--font-example-screen: 'Courier Prime', monospace`, `--font-example-lyrics: 'Caveat', cursive`
  - Spacing, radius, etc.
- Add Google Fonts link in `index.html` for all 6 fonts (see `palette-v3.html` line 7-8 for the exact URL)
- Verify: component with `className="bg-poppy text-white font-heading"` renders correctly

**Commit:** `feat(frontend): add Tailwind CSS with Herbarium design tokens`

---

## Task 3: shadcn/ui Setup

**Goal:** shadcn/ui installed and themed to match Herbarium.

**Context:** shadcn/ui generates component source code into `src/components/ui/`. We customize the CSS variables it uses to point to Herbarium tokens. This gives us accessible, well-tested component primitives (Dialog, Dropdown, Tooltip, etc.) with our visual style.

**What to do:**
- Run `npx shadcn@latest init` — select the options: TypeScript, CSS variables style, use `src/components/ui/` as components path
- Map shadcn's CSS variable convention to Herbarium tokens (in globals.css or the shadcn CSS layer). Key mappings:
  - `--background` → `var(--bg-page)`
  - `--foreground` → `var(--text-primary)`
  - `--primary` → `var(--accent)` (poppy)
  - `--primary-foreground` → white
  - `--secondary` → `var(--bg-surface)` (parchment)
  - `--muted` → `var(--linen)`
  - `--muted-foreground` → `var(--text-secondary)`
  - `--border` → `var(--border)` from Herbarium
  - `--destructive` → `var(--danger)` (poppy)
  - `--ring` → `var(--focus-ring)` (poppy)
  - `--radius` → `var(--radius-md)` (10px)
- Install initial set of shadcn components that we'll need across all phases: `button`, `input`, `dialog`, `dropdown-menu`, `toast`/`sonner`, `skeleton`, `badge`, `tooltip`, `separator`, `card`
- Verify: render a `<Button>` — it should appear with poppy accent color, Space Grotesk font, correct radius

**Commit:** `feat(frontend): setup shadcn/ui with Herbarium theme mapping`

---

## Task 4: Base UI Components (Herbarium-specific)

**Goal:** Custom components that match `palette-v3.html` but don't exist in shadcn.

**Context:** The design system defines several domain-specific components. Implement them as React components referencing the CSS/HTML patterns from `palette-v3.html`. For each component, look at the corresponding section in the HTML file for exact styles.

**Components to create in `src/components/ui/`:**

| Component | Reference in palette-v3.html | Purpose |
|-----------|------------------------------|---------|
| `StatusPill` | "Status Pills — Semantic Axis" section | Shows card state: New (poppy), Learning (goldenrod), Review (cornflower), Mastered (thyme). Dot + label. |
| `SourceTag` | "Source Tags — Categorical Layer" section | Tags for content source: Book (lavender), Screen (teal), Lyrics (heather) |
| `SrsButtons` | "SRS Flashcard" + "SRS Buttons — Disabled" sections | Four grade buttons (Again/Hard/Good/Easy) with semantic colors. Accept interval labels. Support disabled state. |
| `Toast` variants | "Toasts" section (if present) | Success (thyme border), Warning (goldenrod border), Error (poppy border) — or configure shadcn's Sonner with these colors |
| `BotanicalDivider` | "Botanical divider" class `.divider` | Gradient divider (sage → dried-rose → sage) |

**Note:** Don't build Flashcard, VocabCard, or Modal now — they depend on domain data and will be built in later phases. shadcn's `Dialog` covers modal needs.

**Commit:** `feat(frontend): add Herbarium-specific UI components (StatusPill, SourceTag, SrsButtons)`

---

## Task 5: React Router + Layouts

**Goal:** All routes defined with layouts and stub pages.

**Context:** App has two layout zones: (1) Auth pages (login/register) — no navigation, centered layout. (2) Main app — tab navigation (Dictionary, Study, Topics, Inbox) + secondary pages (Settings, Admin). Tab Navigation component reference in `palette-v3.html` section "Tab Navigation". On mobile (<900px) the sidebar hides (see `@media` rule in palette HTML).

**What to do:**
- Install `react-router` v7
- Create two layouts:
  - `AuthLayout` — centered, minimal (for login/register pages)
  - `MainLayout` — top/tab navigation + content area. Navigation items: Dashboard, Dictionary, Study, Topics, Inbox, Settings. Show "Admin" only for admin role (can be hidden for now).
- Create stub pages (just a heading with page name) in `src/pages/`:
  - `LoginPage`, `RegisterPage`
  - `DashboardPage`, `DictionaryPage`, `DictionaryEntryPage`
  - `StudyPage`
  - `TopicsPage`, `InboxPage`
  - `SettingsPage`
  - `AdminPage`
  - `NotFoundPage`
- Define routes:
  - `/login`, `/register` → AuthLayout
  - `/` → redirect to `/dashboard`
  - `/dashboard`, `/dictionary`, `/dictionary/:id`, `/study`, `/topics`, `/inbox`, `/settings`, `/admin` → MainLayout
  - `*` → NotFoundPage
- Navigation should highlight current route (active tab style from `palette-v3.html`: `color: var(--accent); border-bottom-color: var(--accent)`)
- Responsive: on mobile, consider bottom tab bar (height from `--tab-bar-height: 56px`) or hamburger menu

**Commit:** `feat(frontend): add React Router with layouts and page stubs`

---

## Task 6: Apollo Client Setup

**Goal:** Apollo Client configured with auth headers and automatic token refresh.

**Context:** The API uses hybrid REST + GraphQL. All GraphQL goes to `POST /query`. Auth uses JWT access tokens (15 min TTL) in `Authorization: Bearer` header. When access token expires, client must call `POST /auth/refresh` with the refresh token to get new tokens. See `backend_v4/docs/API.md` for full auth flow.

**What to do:**
- Install `@apollo/client` and `graphql`
- Create `src/lib/apollo.ts`:
  - `HttpLink` pointing to GraphQL endpoint (use env var `VITE_API_URL`)
  - `AuthLink` — reads access token from storage, sets `Authorization: Bearer` header
  - Error link — intercept 401 errors, attempt token refresh via `POST /auth/refresh`, retry the failed request. If refresh fails → redirect to `/login`
  - Compose links: authLink → errorLink → httpLink
  - Configure `InMemoryCache` with sensible defaults (type policies for cursor-based pagination on `dictionary` query)
- Create `src/providers/ApolloProvider.tsx` — wraps app with `<ApolloProvider>`
- Create `src/lib/api.ts` — helper for REST calls (auth endpoints). Simple `fetch` wrapper that reads the same token from storage and handles the base URL.
- Add `.env` file with `VITE_API_URL=http://localhost:8080` (or whatever backend port is per `backend_v4/docs/QUICKSTART.md`)
- Add `.env` to `.gitignore`, create `.env.example` with the variable name

**Commit:** `feat(frontend): configure Apollo Client with JWT auth and token refresh`

---

## Task 7: Auth Infrastructure

**Goal:** Token storage, auth context, protected routes — all the wiring so Phase 1 (Auth pages) just plugs in UI.

**Context:** JWT flow per `docs/business/WORKFLOWS.md` and `backend_v4/docs/API.md`:
- Login/Register → receive `{ accessToken, refreshToken, user }`
- Store both tokens (localStorage or in-memory + httpOnly consideration)
- Access token → every request header
- Refresh token → used when access token expires
- Logout → `POST /auth/logout` (revokes all sessions), clear local tokens

**What to do:**
- Create `src/lib/auth.ts` — token storage utilities:
  - `getAccessToken()`, `setAccessToken()`, `getRefreshToken()`, `setRefreshToken()`, `clearTokens()`
  - Store in `localStorage` for now (can switch to more secure approach later)
- Create `src/providers/AuthProvider.tsx`:
  - React context with: `user: User | null`, `isAuthenticated: boolean`, `isLoading: boolean`, `login(tokens, user)`, `logout()`
  - On mount: check if tokens exist, validate by calling `me` GraphQL query, set user
  - `login()` — stores tokens, sets user
  - `logout()` — calls `POST /auth/logout`, clears tokens, redirects to `/login`
- Create `src/components/ProtectedRoute.tsx`:
  - If authenticated → render children
  - If loading → show skeleton/spinner
  - If not authenticated → redirect to `/login`
- Create `src/types/auth.ts`:
  - `User` type: `{ id, email, username, name, avatarUrl, role }`
  - `AuthTokens` type: `{ accessToken, refreshToken }`
- Wire into App.tsx: wrap with AuthProvider, protect MainLayout routes

**Commit:** `feat(frontend): add auth context, token storage, and protected routes`

---

## Task 8: ESLint + Prettier

**Goal:** Consistent code formatting and linting from day one.

**What to do:**
- Configure ESLint with TypeScript + React rules (Vite template may already include basics)
- Install Prettier, create `.prettierrc` with project conventions (single quotes, semicolons, 2-space indent, trailing commas)
- Add npm scripts: `lint`, `lint:fix`, `format`
- Verify: run `npm run lint` — no errors on current codebase

**Commit:** `feat(frontend): configure ESLint and Prettier`

---

## Task 9: Smoke Test & Verify

**Goal:** Everything works together end-to-end.

**What to do:**
- Run `npm run dev` — app starts without errors
- Navigate to `/login` — see AuthLayout with stub
- Navigate to `/dashboard` — redirected to `/login` (not authenticated)
- Manually set a fake token → can see MainLayout with tab navigation and Dashboard stub
- Navigation highlights current route
- Herbarium styles visible: fonts, colors, spacing
- Base components render correctly: Button (poppy), StatusPill, SourceTag
- Apollo Client initialized (can check in React DevTools or console)
- `npm run build` — production build succeeds
- `npm run lint` — passes

**Commit:** `chore(frontend): verify Phase 0 foundation works end-to-end`

---

## Summary

| Task | Description | Key files |
|------|-------------|-----------|
| 1 | Project scaffold | `frontend-real/` structure |
| 2 | Tailwind + Herbarium tokens | `src/styles/herbarium.css`, `src/styles/globals.css` |
| 3 | shadcn/ui setup | `src/components/ui/`, theme mapping |
| 4 | Custom UI components | `StatusPill`, `SourceTag`, `SrsButtons` |
| 5 | Router + layouts | `src/pages/`, `src/layouts/`, route config |
| 6 | Apollo Client | `src/lib/apollo.ts`, `src/providers/ApolloProvider.tsx` |
| 7 | Auth infrastructure | `src/providers/AuthProvider.tsx`, `src/lib/auth.ts` |
| 8 | ESLint + Prettier | Config files |
| 9 | Smoke test | Verification checklist |

**Total:** 9 tasks, ~9 commits
