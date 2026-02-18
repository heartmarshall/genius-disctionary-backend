# Test Frontend Design — MyEnglish Backend v4

**Date:** 2026-02-18
**Stack:** TypeScript + React + Vite + Tailwind CSS
**Location:** `frontend/` in repo root

## Goal

A hybrid test frontend that combines a Mini App (real UX with navigation) and an API Explorer (raw access to every query/mutation). The goal is to test 100% of backend functionality.

## Architecture

```
frontend/
├── src/
│   ├── main.tsx
│   ├── App.tsx                  # Router + Layout
│   ├── api/
│   │   ├── client.ts            # Fetch-based GraphQL client (no Apollo)
│   │   ├── queries.ts           # All GQL queries
│   │   └── mutations.ts         # All GQL mutations
│   ├── auth/
│   │   ├── AuthProvider.tsx     # Context: token storage, refresh, logout
│   │   ├── LoginPage.tsx        # Google OAuth + manual JWT input
│   │   └── useAuth.ts
│   ├── components/
│   │   ├── Layout.tsx           # Sidebar nav + content area
│   │   ├── JsonViewer.tsx       # Pretty-print JSON
│   │   ├── RawPanel.tsx         # Shows GQL query + raw response
│   │   ├── FormField.tsx        # Universal form field
│   │   └── StatusBadge.tsx      # Status badges (NEW, LEARNING, etc)
│   ├── pages/
│   │   ├── DictionaryPage.tsx   # List + search + CRUD
│   │   ├── EntryDetailPage.tsx  # Entry detail (senses, translations, examples, images)
│   │   ├── StudyPage.tsx        # Dashboard + queue + review
│   │   ├── InboxPage.tsx        # Quick notes
│   │   ├── TopicsPage.tsx       # Topics + entry linking
│   │   ├── ProfilePage.tsx      # Profile + SRS settings
│   │   ├── CatalogPage.tsx      # Reference catalog search
│   │   └── ExplorerPage.tsx     # API Explorer — all queries by domain tabs
│   └── hooks/
│       ├── useGraphQL.ts        # Generic query/mutation hook with raw response
│       └── useRawPanel.ts       # RawPanel visibility
├── index.html
├── package.json
├── tsconfig.json
├── vite.config.ts
└── tailwind.config.js
```

## Key Decisions

- **No Apollo/urql** — simple fetch-based client. No cache or normalization needed for a test frontend.
- **React Router** for Mini App navigation + `/explorer` route for API Explorer.
- **RawPanel** — collapsible panel on every page showing last GQL query and raw response.
- **Minimal deps** — react, react-dom, react-router-dom, tailwindcss. Nothing else.

## Auth Flow

Two modes:
1. **Google OAuth** — real OAuth flow with redirect
2. **Manual JWT** — paste a pre-obtained JWT token (for testing without OAuth setup)

Token stored in localStorage, auto-added as `Authorization: Bearer` header.
On 401: attempt refresh, on failure redirect to login.

## Pages

### Auth (LoginPage)
- Google OAuth button + JWT paste field
- Refresh Token + Logout buttons
- Current user info display

### Dictionary (DictionaryPage + EntryDetailPage)
- Entry list with filters (search, hasCard, partOfSpeech, topicID, status), sorting, pagination
- Actions: create from catalog, create custom, batch delete, import, export
- Soft-deleted entries tab with Restore
- Entry detail: Senses CRUD+reorder, Translations CRUD+reorder, Examples CRUD+reorder, UserImages add/delete, Notes edit, Card create/delete

### Study (StudyPage)
- Dashboard stats (due, new, streak, status counts, overdue, active session)
- Study queue list
- Review flow: show card → 4 grade buttons → next
- Session management: Start/Finish/Abandon, session results
- Card history and stats
- Undo last review
- Batch create cards

### Catalog (CatalogPage)
- Search catalog (searchCatalog)
- Preview entry (previewRefEntry) — full RefEntry tree
- "Add to dictionary" button → createEntryFromCatalog with sense selection

### Inbox (InboxPage)
- List with pagination
- Create form (text + context)
- Delete one + Clear all

### Topics (TopicsPage)
- List with entry counts
- CRUD topics
- Link/unlink entries, batch link

### Profile (ProfilePage)
- Current profile (me query)
- SRS settings form: NewCardsPerDay, ReviewsPerDay, MaxIntervalDays, Timezone

### API Explorer (ExplorerPage)
- Tabs: Auth | Dictionary | Content | Study | Inbox | Topics | User | RefCatalog
- Each tab: list of all queries/mutations for that domain
- Each operation: input form + Execute button + JSON result
- Covers 100% of API operations

## GraphQL Client

```typescript
async function graphqlRequest<T>(query: string, variables?: Record<string, unknown>): Promise<{
  data: T | null;
  errors: GraphQLError[] | null;
  raw: { query: string; variables: unknown; response: unknown };
}>
```

## Error Handling

- GraphQL errors with `extensions.code` → toast notification
- Validation errors → field highlighting
- Network errors → top banner

## RawPanel

- Collapsible panel on every page
- Shows: GraphQL query text, variables, HTTP status, response JSON
- "Copy as cURL" button
