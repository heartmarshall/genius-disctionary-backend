# Phase 1: Auth — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Complete authentication cycle — registration, email/password login, Google OAuth login, token management, logout. After this phase, users can create accounts and access the protected app.

**Architecture:** All auth endpoints are REST (`/auth/*`). Apollo Client's auth link (from Phase 0) handles JWT injection. AuthProvider (from Phase 0) manages state. This phase adds the UI pages, form logic, OAuth integration, and wires everything together.

**Key references:**
- API endpoints: `backend_v4/docs/API.md` — Auth section (lines 13–35)
- Auth workflows: `docs/business/WORKFLOWS.md` — Registration, Login, OAuth, Token Refresh, Logout
- Business rules: `docs/business/BUSINESS_RULES.md` — validation limits, rate limiting
- Design components: `frontend-real/design-docs/palette-v3.html` — Input Fields, Buttons, Toasts
- Phase 0 infra: `src/providers/AuthProvider.tsx`, `src/lib/auth.ts`, `src/lib/api.ts`

---

## Task 1: Auth API Layer

**Goal:** Type-safe functions for all REST auth endpoints, used by UI components.

**What to do:**
- Create `src/lib/api/auth.ts` with functions for each auth endpoint:
  - `register(email, username, password)` → `POST /auth/register`
  - `loginPassword(email, password)` → `POST /auth/login/password`
  - `loginOAuth(provider, code)` → `POST /auth/login` with `{ provider: "google", code }`
  - `refreshToken(refreshToken)` → `POST /auth/refresh`
  - `logout(accessToken)` → `POST /auth/logout`
- All functions should use the REST helper from Phase 0 (`src/lib/api.ts`)
- Define response types in `src/types/auth.ts` (extend what Phase 0 created):
  - `AuthResponse`: `{ accessToken, refreshToken, user: User }`
  - `ValidationError`: `{ error, code: "VALIDATION", fields: { field, message }[] }`
- Handle error responses:
  - `400` → parse validation errors, return structured field-level errors
  - `401` → generic "Invalid credentials" (backend hides enumeration details)
  - `429` → rate limited, extract `Retry-After` header if present

**Commit:** `feat(auth): add REST API layer for auth endpoints`

---

## Task 2: Form Validation Utilities

**Goal:** Client-side validation matching backend rules, reusable across auth forms.

**Context:** Validation rules from `docs/business/BUSINESS_RULES.md`:
- Email: valid format, max 254 chars, normalized (lowercased, trimmed)
- Username: 2–50 chars
- Password: 8–72 chars
- All fields required

**What to do:**
- Create `src/lib/validation/auth.ts` with validation functions:
  - `validateEmail(value)` → error string or null
  - `validateUsername(value)` → error string or null
  - `validatePassword(value)` → error string or null
- Each returns a user-friendly error message in English (app is for English learners)
- Create a generic `useFormValidation` hook or use a lightweight form library (react-hook-form recommended — pairs well with shadcn/ui inputs). Decide which approach and stick with it.
- Validation should run on blur + on submit, not on every keystroke

**Commit:** `feat(auth): add client-side form validation utilities`

---

## Task 3: Register Page

**Goal:** Working registration form — user can create an account.

**Context:** Registration is `POST /auth/register` with `{ email, username, password }`. On success → store tokens, set user in AuthProvider, redirect to `/dashboard`. See Input Fields and Buttons in `palette-v3.html` for visual reference.

**What to do:**
- Create `src/pages/RegisterPage.tsx` (replace stub from Phase 0)
- Form fields: email, username, password, confirm password (confirm is client-only)
- Use Input components styled per Herbarium (underline style, error state with poppy color)
- "Create Account" button (primary/poppy)
- Link to login page: "Already have an account? Sign in"
- On submit:
  - Client-side validation first
  - Call `register()` from auth API layer
  - On success → `authContext.login(tokens, user)` → redirect to `/dashboard`
  - On error → show field-level errors (map `ValidationError.fields` to form fields), or toast for generic errors (rate limit, server error)
- Uses AuthLayout (centered, minimal)
- Responsive: works on mobile (full-width inputs) and desktop (max-width card)

**Commit:** `feat(auth): implement registration page`

---

## Task 4: Login Page (Email + Password)

**Goal:** Working login form — user can sign in with email and password.

**Context:** Login is `POST /auth/login/password` with `{ email, password }`. Same response format as register. Backend returns generic "unauthorized" for all failure types (wrong email, wrong password, no account) — so the UI should show a single generic error, not field-specific.

**What to do:**
- Create `src/pages/LoginPage.tsx` (replace stub from Phase 0)
- Form fields: email, password
- "Sign In" button (primary/poppy)
- Link to register: "Don't have an account? Create one"
- On submit:
  - Basic client validation (non-empty fields)
  - Call `loginPassword()`
  - On success → `authContext.login()` → redirect to `/dashboard`
  - On `401` → show generic error "Invalid email or password" (NOT field-specific, per enumeration prevention)
  - On `429` → "Too many attempts. Please try again later."
- Remember: do NOT show separate errors for "email not found" vs "wrong password"

**Commit:** `feat(auth): implement login page`

---

## Task 5: Google OAuth Integration

**Goal:** "Sign in with Google" button that completes the OAuth flow.

**Context:** The backend expects `POST /auth/login` with `{ provider: "google", code: "<auth_code>" }`. The frontend is responsible for getting the authorization code from Google's OAuth consent screen. On success, backend either creates a new account (if email is new), links to existing account (if email matches), or logs in returning user. See `docs/business/WORKFLOWS.md` — OAuth Login flow and `docs/business/INTEGRATIONS.md` — Google OAuth.

**What to do:**
- Install Google OAuth library — `@react-oauth/google` or implement manually with `google-accounts` API
- Add `VITE_GOOGLE_CLIENT_ID` to `.env.example`
- Create `src/components/auth/GoogleSignInButton.tsx`:
  - Renders Google-branded sign-in button (or custom styled to match Herbarium)
  - On click → opens Google consent, receives auth code
  - Calls `loginOAuth("google", code)`
  - On success → `authContext.login()` → redirect to `/dashboard`
  - On error → toast with error message
- Add GoogleSignInButton to both LoginPage and RegisterPage (with a visual divider "or")
- Wrap app with Google OAuth provider (needs client ID from env)

**Commit:** `feat(auth): add Google OAuth sign-in`

---

## Task 6: Token Refresh Integration

**Goal:** Seamless token refresh — user never sees expired token errors during normal use.

**Context:** Access token lives 15 minutes. When it expires, Apollo's error link (from Phase 0) should automatically call `POST /auth/refresh` with the stored refresh token, get new tokens, and retry the failed request. Refresh token itself rotates on every refresh (old one invalidated). If refresh fails (token expired, revoked) → force logout.

**What to do:**
- Verify and complete the Apollo error link logic in `src/lib/apollo.ts` (started in Phase 0):
  - On 401 GraphQL error → attempt refresh
  - On 401 REST error → attempt refresh
  - Prevent multiple simultaneous refresh attempts (queue pending requests while refreshing)
  - On successful refresh → update stored tokens via `authContext` or `auth.ts` storage → retry original request
  - On failed refresh → `authContext.logout()` → redirect to `/login`
- Also handle REST calls refresh (the `src/lib/api.ts` helper should have the same retry logic)
- Test scenarios to verify:
  - Normal request with valid token → works
  - Request with expired access token → refreshes → retries → works
  - Request with expired refresh token → redirects to login
  - Multiple concurrent requests during refresh → all queue and succeed after refresh

**Commit:** `feat(auth): wire up automatic token refresh with retry queue`

---

## Task 7: Logout

**Goal:** User can log out, all sessions are revoked.

**Context:** `POST /auth/logout` requires Bearer token. It revokes ALL refresh tokens for the user (all devices). After logout, clear local state and redirect to login.

**What to do:**
- Add logout button/link in MainLayout navigation (e.g., in user dropdown or settings area)
- On click:
  - Call `POST /auth/logout` with current access token
  - `authContext.logout()` — clear tokens from storage, clear user state
  - Clear Apollo Client cache (`client.clearStore()`)
  - Redirect to `/login`
- Should work even if the logout API call fails (network error) — still clear local state
- Add visual element: user avatar/name in navigation header with dropdown containing "Log out"

**Commit:** `feat(auth): implement logout with session revocation`

---

## Task 8: Auth UX Polish

**Goal:** Handle edge cases and improve the auth experience.

**What to do:**
- **Loading states**: disable submit button + show spinner during API calls (prevent double-submit)
- **Redirect after login**: if user tried to access `/dictionary` but was unauthenticated, redirect back there after login (not always to `/dashboard`). Store intended path before redirect to `/login`.
- **Already authenticated**: if user navigates to `/login` or `/register` while already logged in → redirect to `/dashboard`
- **Password visibility toggle**: eye icon on password fields
- **Toast notifications**: show success toast on registration ("Account created!"), show error toasts for server/network errors
- **Rate limit UX**: if 429 received, disable the form for the `Retry-After` duration with a countdown or message

**Commit:** `feat(auth): add loading states, redirects, and UX polish`

---

## Summary

| Task | Description | Key files |
|------|-------------|-----------|
| 1 | Auth REST API layer | `src/lib/api/auth.ts`, `src/types/auth.ts` |
| 2 | Form validation | `src/lib/validation/auth.ts` |
| 3 | Register page | `src/pages/RegisterPage.tsx` |
| 4 | Login page | `src/pages/LoginPage.tsx` |
| 5 | Google OAuth | `src/components/auth/GoogleSignInButton.tsx` |
| 6 | Token refresh | `src/lib/apollo.ts` (update) |
| 7 | Logout | MainLayout update, AuthProvider wiring |
| 8 | Auth UX polish | Loading, redirects, edge cases |

**Total:** 8 tasks, ~8 commits
