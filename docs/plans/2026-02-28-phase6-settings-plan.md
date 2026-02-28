# Phase 6: Settings & Profile — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** User can view and edit their profile, and configure study parameters that affect how FSRS-5 schedules cards. Compact phase — one page with two sections.

**Architecture:** Profile and settings are separate GraphQL queries (`me`, `userSettings`) and mutations (`updateProfile`, `updateSettings`). Settings directly affect the SRS algorithm behavior, so each parameter needs a clear explanation for the user.

**Key references:**
- API: `backend_v4/docs/API.md` — User section (lines 186–192)
- Business rules: `docs/business/BUSINESS_RULES.md` — settings validation ranges (newCardsPerDay 1-999, reviewsPerDay 1-9999, maxIntervalDays 1-36500, desiredRetention 0.70-0.99, timezone valid IANA, profile name max 255 chars, avatar URL max 512 chars)
- FSRS-5 parameters: `docs/business/BUSINESS_RULES.md` — algorithm parameters table, which are user-adjustable vs server-only
- Design: `frontend-real/design-docs/palette-v3.html` — Input Fields, Buttons

---

## Task 1: Settings GraphQL Layer

**Goal:** Queries and mutations for profile and settings.

**What to do:**
- Create `src/graphql/queries/user.ts`:
  - `ME_QUERY` — user profile (id, email, username, name, role)
  - `USER_SETTINGS_QUERY` — study settings (newCardsPerDay, reviewsPerDay, maxIntervalDays, desiredRetention, timezone)
- Create `src/graphql/mutations/user.ts`:
  - `UPDATE_PROFILE` — update name (avatar via URL)
  - `UPDATE_SETTINGS` — update study parameters
- Define types in `src/types/user.ts` (extend existing `User` from Phase 0):
  - `UserSettings`: `{ newCardsPerDay, reviewsPerDay, maxIntervalDays, desiredRetention, timezone }`

**Commit:** `feat(settings): add GraphQL queries, mutations, and types`

---

## Task 2: Profile Section

**Goal:** View and edit user profile information.

**Context:** `me` query returns id, email, username, name, role. Only `name` is editable via `updateProfile`. Email and username are read-only (set at registration). Avatar URL is optional.

**What to do:**
- Create `src/components/settings/ProfileSection.tsx`:
  - **Display**: avatar (image from URL or initials placeholder), name, email (read-only, muted), username (read-only, muted), role badge (user/admin)
  - **Edit name**: inline edit — click edit icon → name becomes input → save/cancel. Validation: required, max 255 chars.
  - **Avatar**: show current avatar or initials. "Change avatar" → input for URL (max 512 chars, must be valid http/https). Preview before save.
  - On save → `updateProfile` mutation → update AuthProvider user context so navigation reflects new name/avatar
  - Success toast on save

**Commit:** `feat(settings): implement profile section`

---

## Task 3: Study Settings Section

**Goal:** Configure SRS parameters with clear explanations of what each does.

**Context:** These settings directly affect the FSRS-5 algorithm. Users who don't understand SRS can easily break their learning by setting extreme values. Each parameter needs a short explanation and sensible defaults shown. Per `BUSINESS_RULES.md`, `reviewsPerDay` is informational only (not enforced) — make this clear in the UI.

**What to do:**
- Create `src/components/settings/StudySettingsSection.tsx`:
  - **New cards per day** (1–999, default 20):
    - Number input with stepper buttons
    - Explanation: "How many new words to introduce each day. Start with 10-20 if you're new."
  - **Reviews per day goal** (1–9,999, default 200):
    - Number input
    - Explanation: "Your daily review goal. This is informational — all due cards are always shown regardless."
    - Visual hint that this is a soft goal, not a hard limit
  - **Maximum interval** (1–36,500 days, default 365):
    - Number input, show in days. Maybe also display in human-readable format ("~1 year", "~100 years")
    - Explanation: "Longest time between reviews. 365 days (1 year) works well for most learners."
  - **Desired retention** (0.70–0.99, default 0.90):
    - Slider or number input with percentage display (70%–99%)
    - Explanation: "Target probability of remembering a word. Higher = more frequent reviews. 90% is recommended."
    - Warning color if set below 0.80 or above 0.95 (extreme values)
  - **Timezone**:
    - Searchable select/dropdown with IANA timezones
    - Auto-detect browser timezone as suggestion
    - Explanation: "Used to calculate your daily streak and 'new today' count."
  - **Save button**: save all settings at once → `updateSettings` mutation
  - Client-side validation per `BUSINESS_RULES.md` ranges before submit
  - Success toast on save, error handling for invalid values

**Commit:** `feat(settings): implement study settings with explanations and validation`

---

## Task 4: Settings Page Assembly

**Goal:** Combine profile and settings into one cohesive page.

**What to do:**
- Replace stub `src/pages/SettingsPage.tsx`:
  - Page title: "Settings" (Orelega One heading)
  - Two sections separated by divider (BotanicalDivider from Phase 0):
    - **Profile** — ProfileSection component
    - **Study Settings** — StudySettingsSection component
  - Both sections fetch data in parallel (`me` + `userSettings` queries)
  - Loading: skeleton for both sections
  - Responsive: single column on all breakpoints (settings forms are naturally narrow)
- Consider: should logout be here or stay in navigation? Keep it in navigation header (Phase 1) — settings page is for configuration, not session management.

**Commit:** `feat(settings): assemble settings page with profile and study sections`

---

## Summary

| Task | Description | Key files |
|------|-------------|-----------|
| 1 | GraphQL layer | `src/graphql/queries/user.ts`, `mutations/user.ts`, `src/types/user.ts` |
| 2 | Profile section | `src/components/settings/ProfileSection.tsx` |
| 3 | Study settings | `src/components/settings/StudySettingsSection.tsx` |
| 4 | Settings page | `src/pages/SettingsPage.tsx` |

**Total:** 4 tasks, ~4 commits
