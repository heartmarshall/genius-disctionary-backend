# Phase 2: Dashboard — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Home page after login — user sees their learning progress at a glance, can start studying, and understands what needs attention today. First real screen that uses GraphQL + Apollo.

**Architecture:** Single `dashboard` GraphQL query returns all stats in one request. No mutations. The page is a grid of stat widgets that adapts from single-column (mobile) to multi-column (desktop). This is the first page to integrate Apollo Client with real data.

**Key references:**
- API: `backend_v4/docs/API.md` — `dashboard` query (lines 138–142)
- Dashboard workflow: `docs/business/WORKFLOWS.md` — "Viewing the Dashboard" section (8 metrics loaded in parallel)
- Business rules: `docs/business/BUSINESS_RULES.md` — streak calculation, accuracy formula
- Design: `frontend-real/design-docs/palette-v3.html` — Status Pills, Skeleton loaders, Buttons
- Domain model: `docs/business/DOMAIN_MODEL.md` — Card states, Study Session

---

## Task 1: Dashboard GraphQL Query

**Goal:** Apollo query hook for dashboard data, typed and ready to use.

**What to do:**
- Create `src/graphql/queries/dashboard.ts`:
  - Define `DASHBOARD_QUERY` gql query matching the API:
    ```graphql
    query Dashboard {
      dashboard {
        dueCount
        newCount
        reviewedToday
        newToday
        streak
        overdueCount
        statusCounts { new, learning, review, relearning, total }
        activeSession { id, status }
      }
    }
    ```
  - Export typed result interface `DashboardData`
- Create `src/hooks/useDashboard.ts` — thin wrapper around `useQuery(DASHBOARD_QUERY)` with:
  - `pollInterval` or refetch strategy (dashboard data changes after every review session)
  - Return `{ data, loading, error, refetch }`

**Commit:** `feat(dashboard): add GraphQL query and hook`

---

## Task 2: Stat Widgets

**Goal:** Reusable stat card components for displaying dashboard metrics.

**Context:** Dashboard shows 8 metrics. Each one is a small card with a label, value, and optionally an icon or accent color. These are not in `palette-v3.html` as ready components, so design them using the existing Herbarium tokens (bg-card, border, elevation-1, radius-md).

**What to do:**
- Create `src/components/dashboard/StatCard.tsx`:
  - Props: `label`, `value` (number or string), `accent` (optional color for value), `icon` (optional), `subtitle` (optional secondary text)
  - Visual: white card, subtle border, elevation-1 shadow, label in text-secondary (sm), value large and prominent (lg or xl), optional accent color on value
- Create `src/components/dashboard/StreakDisplay.tsx`:
  - Shows streak as a number with "days" label
  - Could include a flame/fire icon or simple visual indicator
  - Per `BUSINESS_RULES.md`: streak = consecutive days with at least one review, counts backward from today (or yesterday)
- Create `src/components/dashboard/StatusDistribution.tsx`:
  - Shows card counts by state: New, Learning, Review, Relearning
  - Use StatusPill components from Phase 0 (poppy/goldenrod/cornflower/thyme)
  - Display as horizontal bar or row of pills with counts
  - Total count shown separately

**Commit:** `feat(dashboard): add stat card components`

---

## Task 3: Dashboard Page

**Goal:** Assemble the dashboard page — all widgets, responsive layout, real data.

**What to do:**
- Replace stub `src/pages/DashboardPage.tsx` with full implementation
- Layout structure:
  - **Header**: "Dashboard" title (Orelega One) or greeting ("Good morning, {name}")
  - **Primary action**: "Start Study" button (btn-calm / thyme) — prominent, always visible
    - If `activeSession` exists → label "Continue Study" instead
    - Links to `/study` (stub until Phase 4)
  - **Stats grid**: responsive grid of StatCards
    - Due Count — "Cards due" (accent: poppy if overdue > 0)
    - New Count — "New cards available"
    - Reviewed Today — "Reviewed today"
    - New Today — "New today"
    - Overdue Count — "Overdue" (accent: poppy, hidden if 0)
    - Streak — StreakDisplay component
  - **Status distribution**: StatusDistribution component below stats
- Responsive:
  - Mobile: single column, study button at top
  - Tablet: 2-column grid
  - Desktop: 3 or 4-column grid
- Fetch data with `useDashboard()` hook

**Commit:** `feat(dashboard): implement dashboard page with stats grid`

---

## Task 4: Loading & Empty States

**Goal:** Polish the dashboard for all data conditions.

**What to do:**
- **Loading state**: while `useDashboard()` is loading, show skeleton grid matching the stat card layout. Use Skeleton components from Phase 0 (per `palette-v3.html` skeleton styles — pulse animation, straw color).
- **Empty state** (new user, no cards at all):
  - When `statusCounts.total === 0` → show a welcoming message instead of stats
  - Message like "Your dictionary is empty. Start by adding some words!" with a button linking to `/dictionary`
  - Still show the basic structure, just replace the stat grid with the welcome CTA
- **Error state**: if query fails → show error message with retry button
- **Zero values**: most stats will be 0 for new users — that's fine, show 0. Only hide overdue if it's 0.

**Commit:** `feat(dashboard): add loading skeletons and empty state`

---

## Summary

| Task | Description | Key files |
|------|-------------|-----------|
| 1 | GraphQL query + hook | `src/graphql/queries/dashboard.ts`, `src/hooks/useDashboard.ts` |
| 2 | Stat widgets | `src/components/dashboard/StatCard.tsx`, `StreakDisplay.tsx`, `StatusDistribution.tsx` |
| 3 | Dashboard page | `src/pages/DashboardPage.tsx` |
| 4 | Loading & empty states | Skeletons, welcome message, error handling |

**Total:** 4 tasks, ~4 commits
