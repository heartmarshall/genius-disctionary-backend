# Phase 8: Polish & Edge Cases — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Take the functional app from Phases 0–7 and bring it to production quality — consistent error handling, loading states audit, animations, accessibility, responsive audit, performance optimizations. No new features — only refinement of what exists.

**Architecture:** This phase is horizontal — touches all existing pages and components. Work is organized by concern (errors, a11y, performance) rather than by feature domain.

**Key references:**
- Design system: `frontend-real/design-docs/palette-v3.html` — motion tokens (duration-fast/normal/slow, ease-spring/standard), skeleton styles, toast patterns, elevation levels, focus ring
- Business rules: `docs/business/BUSINESS_RULES.md` — all validation limits (for error messages)
- Open questions: `docs/business/UNKNOWNS.md` — missing features to consider (no forgot password, no email verification, no session timeout)
- All phase plans: `docs/plans/2026-02-28-phase*-plan.md` — verify nothing was missed

---

## Task 1: Global Error Handling

**Goal:** Consistent, user-friendly error handling across the entire app.

**What to do:**
- **Error Boundary**: create `src/components/ErrorBoundary.tsx`
  - Wraps the app (or per-route) to catch React render errors
  - Shows friendly error page: "Something went wrong" with "Try again" (reload) and "Go home" buttons
  - Styled per Herbarium: poppy accent for error indicator, parchment background
  - Log errors to console (production logging integration can come later)
- **Apollo error handling audit**: review all GraphQL queries/mutations across phases
  - Every `useQuery` should handle `error` state — show inline error with retry button, not blank screen
  - Every `useMutation` should have `onError` → toast with user-friendly message
  - Map common GraphQL error codes to messages: VALIDATION → field errors, UNAUTHORIZED → redirect to login, NOT_FOUND → "Item not found", generic → "Something went wrong. Please try again."
- **REST error handling audit**: review auth and admin API calls
  - 429 → "Too many requests" with retry timing
  - 403 → "Access denied"
  - 500 → "Server error. Please try again later."
  - Network error (fetch failed) → "Connection error. Check your internet."
- **Toast consistency**: ensure all mutations show success/error toasts. Audit every mutation call across phases 1–7. Toast styles per `palette-v3.html`: success (thyme border), error (poppy border), warning (goldenrod border).

**Commit:** `fix(ui): add global error boundary and standardize error handling`

---

## Task 2: Loading States Audit

**Goal:** Every data-fetching view has proper loading state — no blank screens, no layout shift.

**What to do:**
- Audit every page and component that fetches data. For each, verify:
  - Initial load → skeleton or spinner shown
  - Skeleton matches the shape of the loaded content (prevents layout shift)
  - Refetch/background update → does NOT show skeleton (use Apollo's `networkStatus` or `previousData` to keep stale content visible during refetch)
- **Pages to audit**:
  - Dashboard: stat card skeletons ✓ (Phase 2 Task 4)
  - Dictionary list: vocab card skeletons ✓ (Phase 3 Task 10)
  - Dictionary entry: content block skeletons ✓ (Phase 3 Task 10)
  - Study: flashcard skeleton ✓ (Phase 4 Task 7)
  - Topics: topic card skeletons ✓ (Phase 5 Task 5)
  - Inbox: list item skeletons ✓ (Phase 5 Task 5)
  - Settings: section skeletons ✓ (Phase 6 Task 4)
  - Admin users: table row skeletons ✓ (Phase 7 Task 5)
  - Admin enrichment: stat + list skeletons ✓ (Phase 7 Task 5)
- **Button loading states**: every submit button across the app should show spinner/disabled state during API call. Audit: register, login, add word, review card, save settings, change role, etc.
- **Skeleton component review**: ensure all skeletons use `palette-v3.html` styles (straw color, pulse animation at 1.8s)

**Commit:** `fix(ui): audit and fix loading states across all pages`

---

## Task 3: Empty States Audit

**Goal:** Every list/collection view has a meaningful empty state — not just blank space.

**What to do:**
- Audit and ensure each view has a helpful empty state message with a CTA:

| View | Empty condition | Message + CTA |
|------|----------------|---------------|
| Dashboard (no cards) | `statusCounts.total === 0` | "Start building your vocabulary!" → Dictionary |
| Dictionary list | No entries | "Your dictionary is empty" → Search catalog / Add word |
| Dictionary search | No results for query | "No words matching '{query}'" → Adjust filters |
| Dictionary by topic | Topic has no entries | "No words in this topic yet" → Add words |
| Trash | No deleted entries | "Trash is empty" |
| Study queue | No due + no new | "All caught up! No cards to review." → Dashboard |
| Topics | No topics | "Create your first topic to organize vocabulary" → New Topic |
| Inbox | No items | "Jot down words you encounter and look them up later" |
| Admin users | No users (edge case) | "No users found" |
| Admin enrichment queue | No items for filter | "No {status} enrichment items" |

- Visual style: centered text, muted color (text-secondary), relevant icon optional, prominent CTA button

**Commit:** `fix(ui): audit and standardize empty states with CTAs`

---

## Task 4: Animations & Transitions

**Goal:** Add subtle motion that makes the app feel polished, using Herbarium motion tokens.

**Context:** Motion tokens from `palette-v3.html`:
- `--duration-fast: 100ms` — micro-interactions (button press, toggle)
- `--duration-normal: 200ms` — standard transitions (hover, focus, page elements)
- `--duration-slow: 350ms` — larger movements (modals, page transitions)
- `--ease-spring: cubic-bezier(0.34, 1.56, 0.64, 1)` — playful bounce for cards, buttons
- `--ease-standard: cubic-bezier(0.25, 0.1, 0.25, 1)` — smooth for general transitions

**What to do:**
- **Page transitions**: subtle fade-in when navigating between pages (React Router + CSS transition or Framer Motion if needed). Duration: slow (350ms), ease: standard.
- **Flashcard reveal**: answer section slides/fades in on reveal. Duration: normal (200ms).
- **SRS button press**: scale down on active (already in `palette-v3.html`: `:active { transform: scale(0.97) }`), hover lift (`translateY(-2px)`)
- **Toast enter/exit**: slide in from top-right, fade out. Duration: normal.
- **Modal open/close**: scale up from 0.95 + fade in. Duration: slow. Overlay fade.
- **List item enter**: stagger fade-in for dictionary/topic/inbox lists on initial load. Subtle, duration: fast per item.
- **Button hover**: all buttons already have hover states in `palette-v3.html` — verify they're applied (translateY, shadow changes)
- Don't over-animate — keep it subtle. Respect `prefers-reduced-motion` media query: disable all non-essential animations.

**Commit:** `feat(ui): add animations and transitions using Herbarium motion tokens`

---

## Task 5: Accessibility (a11y)

**Goal:** App is usable with keyboard, screen readers, and meets WCAG 2.1 AA baseline.

**What to do:**
- **Keyboard navigation**:
  - All interactive elements focusable and operable with keyboard
  - Focus ring visible: poppy color (`--focus-ring`) per `palette-v3.html`
  - Tab order logical on every page
  - Study page: keyboard shortcuts 1-2-3-4 for grades (Phase 4 Task 4), Space/Enter to reveal answer
  - Escape closes modals/dialogs
- **ARIA labels**:
  - Navigation landmarks: `<nav>`, `<main>`, `<aside>`
  - Form inputs: associated `<label>` elements (shadcn/ui handles this, verify)
  - Status pills: `aria-label="Card status: Learning"`
  - SRS buttons: `aria-label="Grade: Again, next review in less than 1 minute"`
  - Progress bar in study: `role="progressbar"` with `aria-valuenow`
  - Toast notifications: `role="status"` or `aria-live="polite"`
  - Loading skeletons: `aria-busy="true"` on parent container
- **Color contrast**: verify all text meets 4.5:1 contrast ratio against backgrounds. Herbarium uses warm muted colors — check `text-tertiary` and `text-disabled` against white and parchment backgrounds specifically.
- **Screen reader testing**: verify key flows are navigable — login, add word, study session, view dashboard
- **`prefers-reduced-motion`**: disable animations for users who prefer reduced motion (Task 4)

**Commit:** `feat(ui): add keyboard navigation, ARIA labels, and a11y improvements`

---

## Task 6: Responsive Audit

**Goal:** All pages work correctly on mobile (360px), tablet (768px), and desktop (1280px+).

**What to do:**
- Test every page at three breakpoints. Key areas to check:
  - **Navigation**: mobile — bottom tab bar or hamburger menu, tablet/desktop — top/side tabs
  - **Dashboard**: stat grid adapts columns (1→2→3-4)
  - **Dictionary list**: full-width cards on mobile, more compact on desktop
  - **Dictionary entry detail**: single column on mobile, could be two-column on desktop (content + sidebar with card/topics info)
  - **Study flashcard**: centered, respects `--container-narrow` (480px), full-width padding on mobile
  - **SRS buttons**: equal width, readable labels on small screens
  - **Forms** (register, login, settings, add word): max-width container, full-width inputs on mobile
  - **Admin tables**: horizontal scroll on mobile, or card-based layout instead of table
  - **Modals/dialogs**: full-screen on mobile, centered with max-width on desktop
- Fix any overflow, truncation, or touch target issues (min 44x44px touch targets)
- Verify text doesn't overflow containers (long word names, long definitions)

**Commit:** `fix(ui): responsive audit fixes across all pages`

---

## Task 7: Optimistic Updates & Performance

**Goal:** Key interactions feel instant, app loads fast.

**What to do:**
- **Optimistic updates** for frequently used mutations:
  - `reviewCard` — immediately show next card, don't wait for server response (revert on error)
  - `deleteEntry` — immediately remove from list
  - `deleteInboxItem` — immediately remove from list
  - `linkEntryToTopic` — immediately show topic tag on entry
  - Pattern: use Apollo's `optimisticResponse` + cache update
- **Code splitting**: lazy-load pages with `React.lazy()` + `Suspense`
  - Split by route: Dashboard, Dictionary, Study, Topics, Inbox, Settings, Admin
  - Admin pages definitely lazy (most users never visit)
  - Show simple loading spinner during chunk load
- **Bundle audit**: check bundle size, identify large dependencies. Consider:
  - Apollo Client is heavy — verify tree-shaking is working
  - Timezone picker library (if used) can be large — lazy load
  - Google OAuth library — lazy load on auth pages only
- **Image optimization**: avatar URLs should use reasonable sizes, add loading="lazy" to images
- **Cache strategy**: configure Apollo cache `fetchPolicy` per query type:
  - Dashboard: `cache-and-network` (show cached, refetch in background)
  - Dictionary list: `cache-first` (cache persists during session)
  - Study queue: `network-only` (always fresh)
  - Settings: `cache-first`

**Commit:** `perf(ui): add optimistic updates, code splitting, and cache tuning`

---

## Task 8: Final Verification

**Goal:** Everything works together, nothing was missed.

**What to do:**
- **Full flow test** (manual):
  1. Register new account → see dashboard (empty state)
  2. Search catalog → add 3 words → see dictionary populated
  3. Create cards for all words
  4. Start study session → review cards → finish → see results
  5. Go to dashboard → stats updated
  6. Create topic → link words → filter dictionary by topic
  7. Add inbox items → "look up" one → add to dictionary
  8. Change settings → verify new cards per day affects study queue
  9. Logout → login again → data persists
  10. (If admin) Switch to admin panel → view users → view enrichment queue
- **Build verification**: `npm run build` succeeds, no TypeScript errors, no lint errors
- **Console check**: no React warnings, no uncaught errors, no Apollo cache warnings
- **Lighthouse audit** (optional): check performance, accessibility, best practices scores

**Commit:** `chore(frontend): final verification of all flows`

---

## Summary

| Task | Description | Scope |
|------|-------------|-------|
| 1 | Global error handling | ErrorBoundary, Apollo/REST error audit, toast consistency |
| 2 | Loading states audit | Skeleton verification across all pages |
| 3 | Empty states audit | Meaningful messages + CTAs for every list view |
| 4 | Animations | Page transitions, flashcard reveal, button effects, reduced-motion |
| 5 | Accessibility | Keyboard nav, ARIA labels, contrast, screen reader |
| 6 | Responsive audit | Mobile/tablet/desktop verification for all pages |
| 7 | Performance | Optimistic updates, code splitting, cache strategy |
| 8 | Final verification | Full flow manual test, build check, console audit |

**Total:** 8 tasks, ~8 commits
