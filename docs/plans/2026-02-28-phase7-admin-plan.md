# Phase 7: Admin Panel — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Administration panel for managing users and AI enrichment queue. Only accessible to users with role=admin. Uses REST endpoints (not GraphQL).

**Architecture:** All admin endpoints are REST (`/admin/*`). Uses the same REST helper from Phase 1 (`src/lib/api.ts`) with Bearer token. Admin panel is a separate section of the app — either a separate layout or a distinct navigation area. Access is gated by user role from AuthProvider.

**Key references:**
- API: `backend_v4/docs/API.md` — Admin section (lines 37–47)
- Roles: `docs/business/ROLES_AND_ACCESS.md` — permission matrix, admin endpoints list, self-demote protection
- Enrichment: `docs/business/INTEGRATIONS.md` — AI enrichment queue (pending→processing→done/failed), Anthropic Claude integration
- Open questions: `docs/business/UNKNOWNS.md` — "Should enrichment failures auto-retry?" (currently manual)
- Design: `frontend-real/design-docs/palette-v3.html` — Buttons, Input Fields, Status Pills, Tags

---

## Task 1: Admin API Layer

**Goal:** Type-safe functions for all REST admin endpoints.

**What to do:**
- Create `src/lib/api/admin.ts` with functions for each endpoint:
  - `getUsers(limit, offset)` → `GET /admin/users?limit=&offset=`
  - `updateUserRole(userId, role)` → `PUT /admin/users/{id}/role` with `{ role: "admin" | "user" }`
  - `getEnrichmentStats()` → `GET /admin/enrichment/stats`
  - `getEnrichmentQueue(status?, limit?, offset?)` → `GET /admin/enrichment/queue?status=&limit=&offset=`
  - `enqueueEnrichment(refEntryId)` → `POST /admin/enrichment/enqueue`
  - `retryFailedEnrichments()` → `POST /admin/enrichment/retry`
  - `resetProcessingEnrichments()` → `POST /admin/enrichment/reset-processing`
- All functions use the REST helper with Bearer token from auth storage
- Define types in `src/types/admin.ts`:
  - `AdminUser`: `{ id, email, username, name, role, avatarUrl }`
  - `UsersResponse`: `{ users: AdminUser[], total: number }`
  - `EnrichmentStats`: `{ pending, processing, done, failed, total }`
  - `EnrichmentQueueItem`: `{ id, refEntryId, status, createdAt, updatedAt, error? }`

**Commit:** `feat(admin): add REST API layer for admin endpoints`

---

## Task 2: Admin Route Guard

**Goal:** Protect admin pages — only role=admin can access.

**What to do:**
- Create `src/components/AdminRoute.tsx`:
  - Reads user role from AuthProvider
  - If `role === "admin"` → render children
  - If `role === "user"` → show "Access Denied" page or redirect to `/dashboard` with toast "You don't have admin access"
  - If loading → show spinner/skeleton
- Wire into router: wrap `/admin` and `/admin/*` routes with AdminRoute
- Add "Admin" link in MainLayout navigation — only visible when `user.role === "admin"`
- Consider: separate admin layout or reuse MainLayout with an admin sub-navigation? Simpler approach: reuse MainLayout, add tab-style sub-nav within the admin page (Users | Enrichment).

**Commit:** `feat(admin): add admin route guard and navigation`

---

## Task 3: User Management Page

**Goal:** List all users, change roles.

**Context:** `GET /admin/users` returns paginated list (offset-based). `PUT /admin/users/{id}/role` changes role. Self-demote is protected by backend (admin cannot change their own role to user). Per `ROLES_AND_ACCESS.md`: all admins have equal privileges, no super-admin concept.

**What to do:**
- Create `src/pages/admin/UsersPage.tsx`:
  - **Users table/list**: columns — avatar (or initials), name, email, username, role (pill/badge), actions
  - **Pagination**: offset-based (prev/next buttons or page numbers), default limit 20
  - **Role toggle**: button or dropdown per user row to change role
    - "Make Admin" for users, "Remove Admin" for admins
    - Confirmation dialog: "Change {name}'s role to {newRole}?"
    - Disable the action for the current user's own row (self-demote protection, but backend also enforces this)
    - On success → update list, toast "Role updated"
    - On error → toast with error message
  - **Search/filter**: optional — simple text search by name or email (client-side filter on current page, or if many users, could be useful)
- Create `src/hooks/useAdminUsers.ts` — manages fetch + pagination state + role update

**Commit:** `feat(admin): implement user management page`

---

## Task 4: Enrichment Queue Page

**Goal:** Monitor and manage the AI enrichment pipeline.

**Context:** When users add words from catalog, the system queues them for AI enrichment (Anthropic Claude). Queue items progress: pending → processing → done or failed. Admins can: view stats, browse queue, retry failed items, reset stuck "processing" items, manually enqueue a word. See `docs/business/INTEGRATIONS.md` for the enrichment pipeline.

**What to do:**
- Create `src/pages/admin/EnrichmentPage.tsx`:
  - **Stats dashboard** (top section):
    - Call `getEnrichmentStats()` → display 5 metrics as stat cards:
    - Pending, Processing, Done, Failed, Total
    - Color-code: pending (goldenrod), processing (cornflower), done (thyme), failed (poppy)
  - **Action buttons** (below stats):
    - "Retry Failed" button (btn-primary) → `retryFailedEnrichments()` → toast "X items retried"
    - "Reset Stuck" button (btn-secondary) → `resetProcessingEnrichments()` → toast "X items reset"
    - Both should refetch stats after action
  - **Queue list** (main section):
    - Call `getEnrichmentQueue(status, limit, offset)`
    - Filter by status: dropdown (all / pending / processing / done / failed)
    - Offset pagination
    - Each row: ref entry ID or word text (if available), status (pill with semantic color), created/updated timestamps, error message (if failed)
  - **Manual enqueue**: "Enqueue Word" button → dialog with ref entry ID input → `enqueueEnrichment(refEntryId)` → toast + refetch
- Create `src/hooks/useEnrichmentQueue.ts` — manages stats fetch, queue fetch with filters/pagination, action calls

**Commit:** `feat(admin): implement enrichment queue management page`

---

## Task 5: Admin Page Assembly + Loading States

**Goal:** Combine admin sub-pages into a cohesive admin section.

**What to do:**
- Replace stub `src/pages/AdminPage.tsx`:
  - Sub-navigation: "Users" | "Enrichment" tabs (reuse tab-nav pattern from `palette-v3.html`)
  - Default to Users tab
  - Routes: `/admin` → Users, `/admin/enrichment` → Enrichment (nested routes or tab-based switching)
- Loading states:
  - Users: skeleton table rows
  - Enrichment stats: skeleton stat cards
  - Enrichment queue: skeleton list items
- Empty states:
  - No users (unlikely but handle): "No users found"
  - Empty queue for selected filter: "No {status} enrichment items"
- Error states: standard error + retry for all API calls

**Commit:** `feat(admin): assemble admin page with tabs and loading states`

---

## Summary

| Task | Description | Key files |
|------|-------------|-----------|
| 1 | REST API layer | `src/lib/api/admin.ts`, `src/types/admin.ts` |
| 2 | Route guard | `src/components/AdminRoute.tsx`, navigation visibility |
| 3 | User management | `src/pages/admin/UsersPage.tsx` |
| 4 | Enrichment queue | `src/pages/admin/EnrichmentPage.tsx` |
| 5 | Assembly + polish | `src/pages/AdminPage.tsx`, tabs, loading/empty states |

**Total:** 5 tasks, ~5 commits
