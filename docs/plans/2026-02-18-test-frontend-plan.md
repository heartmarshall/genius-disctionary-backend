# Test Frontend Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a hybrid test frontend (Mini App + API Explorer) for MyEnglish Backend v4 that covers 100% of GraphQL API functionality.

**Architecture:** React SPA with fetch-based GraphQL client (no Apollo). Mini App pages for common workflows + API Explorer for raw access to every query/mutation. RawPanel on each page shows request/response JSON. Auth via manual JWT paste (backend has no login endpoint exposed).

**Tech Stack:** TypeScript, React 18, React Router v6, Vite, Tailwind CSS v4.

---

### Task 1: Project Scaffold

**Files:**
- Create: `frontend/package.json`
- Create: `frontend/tsconfig.json`
- Create: `frontend/vite.config.ts`
- Create: `frontend/index.html`
- Create: `frontend/src/main.tsx`
- Create: `frontend/src/App.tsx`
- Create: `frontend/src/index.css`

**Step 1: Scaffold Vite project**

```bash
cd /home/alodi/playgorund/myprojects/genius-disctionary-backend
npm create vite@latest frontend -- --template react-ts
```

**Step 2: Install dependencies**

```bash
cd frontend
npm install react-router-dom
npm install -D tailwindcss @tailwindcss/vite
```

**Step 3: Configure Tailwind**

Replace `frontend/src/index.css` with:
```css
@import "tailwindcss";
```

Add Tailwind plugin to `frontend/vite.config.ts`:
```typescript
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: 3000,
    proxy: {
      '/query': 'http://localhost:8080',
      '/health': 'http://localhost:8080',
      '/ready': 'http://localhost:8080',
      '/live': 'http://localhost:8080',
    },
  },
})
```

**Step 4: Create minimal App.tsx with router**

```tsx
// frontend/src/App.tsx
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'

function App() {
  return (
    <BrowserRouter>
      <div className="min-h-screen bg-gray-50 text-gray-900">
        <h1 className="text-2xl p-4">MyEnglish Test Frontend</h1>
        <Routes>
          <Route path="/" element={<div className="p-4">Home</div>} />
        </Routes>
      </div>
    </BrowserRouter>
  )
}

export default App
```

**Step 5: Verify it runs**

```bash
cd frontend && npm run dev
```

Expected: App loads at http://localhost:3000 with "MyEnglish Test Frontend" heading, Tailwind styles applied.

**Step 6: Commit**

```bash
git add frontend/
git commit -m "feat(frontend): scaffold Vite + React + Tailwind project"
```

---

### Task 2: GraphQL Client + Auth Context

**Files:**
- Create: `frontend/src/api/client.ts`
- Create: `frontend/src/api/types.ts`
- Create: `frontend/src/auth/AuthProvider.tsx`
- Create: `frontend/src/auth/useAuth.ts`

**Step 1: Create GraphQL types**

```typescript
// frontend/src/api/types.ts
export interface GraphQLError {
  message: string
  path?: string[]
  extensions?: {
    code?: string
    fields?: { field: string; message: string }[]
  }
}

export interface GraphQLResponse<T> {
  data: T | null
  errors: GraphQLError[] | null
}

export interface GraphQLResult<T> {
  data: T | null
  errors: GraphQLError[] | null
  raw: {
    query: string
    variables: unknown
    response: unknown
    status: number
  }
}
```

**Step 2: Create the GraphQL client**

```typescript
// frontend/src/api/client.ts
import { GraphQLResult } from './types'

let getToken: () => string | null = () => null

export function setTokenGetter(fn: () => string | null) {
  getToken = fn
}

const API_URL = '/query'

export async function graphql<T = unknown>(
  query: string,
  variables?: Record<string, unknown>,
): Promise<GraphQLResult<T>> {
  const token = getToken()
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const body = JSON.stringify({ query, variables })

  const res = await fetch(API_URL, {
    method: 'POST',
    headers,
    body,
  })

  const json = await res.json()

  return {
    data: json.data ?? null,
    errors: json.errors ?? null,
    raw: {
      query,
      variables: variables ?? null,
      response: json,
      status: res.status,
    },
  }
}
```

**Step 3: Create AuthProvider**

```tsx
// frontend/src/auth/AuthProvider.tsx
import { createContext, useState, useCallback, useEffect, ReactNode } from 'react'
import { setTokenGetter } from '../api/client'

export interface AuthContextValue {
  token: string | null
  setToken: (token: string | null) => void
  isAuthenticated: boolean
  logout: () => void
}

export const AuthContext = createContext<AuthContextValue>({
  token: null,
  setToken: () => {},
  isAuthenticated: false,
  logout: () => {},
})

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setTokenState] = useState<string | null>(
    () => localStorage.getItem('jwt_token'),
  )

  const setToken = useCallback((t: string | null) => {
    setTokenState(t)
    if (t) {
      localStorage.setItem('jwt_token', t)
    } else {
      localStorage.removeItem('jwt_token')
    }
  }, [])

  const logout = useCallback(() => setToken(null), [setToken])

  useEffect(() => {
    setTokenGetter(() => token)
  }, [token])

  return (
    <AuthContext.Provider value={{ token, setToken, isAuthenticated: !!token, logout }}>
      {children}
    </AuthContext.Provider>
  )
}
```

**Step 4: Create useAuth hook**

```typescript
// frontend/src/auth/useAuth.ts
import { useContext } from 'react'
import { AuthContext, AuthContextValue } from './AuthProvider'

export function useAuth(): AuthContextValue {
  return useContext(AuthContext)
}
```

**Step 5: Wire AuthProvider into App**

Update `App.tsx` to wrap routes with `<AuthProvider>`.

**Step 6: Commit**

```bash
git add frontend/src/api/ frontend/src/auth/
git commit -m "feat(frontend): add GraphQL client and auth context"
```

---

### Task 3: useGraphQL Hook + RawPanel + JsonViewer

**Files:**
- Create: `frontend/src/hooks/useGraphQL.ts`
- Create: `frontend/src/components/RawPanel.tsx`
- Create: `frontend/src/components/JsonViewer.tsx`

**Step 1: Create useGraphQL hook**

```typescript
// frontend/src/hooks/useGraphQL.ts
import { useState, useCallback } from 'react'
import { graphql } from '../api/client'
import { GraphQLError } from '../api/types'

export interface UseGraphQLState<T> {
  data: T | null
  errors: GraphQLError[] | null
  loading: boolean
  raw: { query: string; variables: unknown; response: unknown; status: number } | null
  execute: (query: string, variables?: Record<string, unknown>) => Promise<T | null>
  reset: () => void
}

export function useGraphQL<T = unknown>(): UseGraphQLState<T> {
  const [data, setData] = useState<T | null>(null)
  const [errors, setErrors] = useState<GraphQLError[] | null>(null)
  const [loading, setLoading] = useState(false)
  const [raw, setRaw] = useState<UseGraphQLState<T>['raw']>(null)

  const execute = useCallback(async (query: string, variables?: Record<string, unknown>) => {
    setLoading(true)
    setErrors(null)
    try {
      const result = await graphql<T>(query, variables)
      setData(result.data)
      setErrors(result.errors)
      setRaw(result.raw)
      return result.data
    } catch (err) {
      const networkError = { message: String(err), extensions: { code: 'NETWORK_ERROR' } }
      setErrors([networkError])
      setRaw(null)
      return null
    } finally {
      setLoading(false)
    }
  }, [])

  const reset = useCallback(() => {
    setData(null)
    setErrors(null)
    setRaw(null)
  }, [])

  return { data, errors, loading, raw, execute, reset }
}
```

**Step 2: Create JsonViewer**

```tsx
// frontend/src/components/JsonViewer.tsx
interface Props {
  data: unknown
  maxHeight?: string
}

export function JsonViewer({ data, maxHeight = '400px' }: Props) {
  const json = JSON.stringify(data, null, 2)

  return (
    <pre
      className="bg-gray-900 text-green-400 text-xs p-3 rounded overflow-auto font-mono"
      style={{ maxHeight }}
    >
      {json}
    </pre>
  )
}
```

**Step 3: Create RawPanel**

```tsx
// frontend/src/components/RawPanel.tsx
import { useState } from 'react'
import { JsonViewer } from './JsonViewer'

interface Props {
  raw: { query: string; variables: unknown; response: unknown; status: number } | null
}

export function RawPanel({ raw }: Props) {
  const [open, setOpen] = useState(false)

  if (!raw) return null

  return (
    <div className="mt-4 border border-gray-300 rounded">
      <button
        onClick={() => setOpen(!open)}
        className="w-full text-left px-3 py-2 bg-gray-100 hover:bg-gray-200 text-sm font-mono flex justify-between"
      >
        <span>Raw Request/Response (HTTP {raw.status})</span>
        <span>{open ? '▼' : '▶'}</span>
      </button>
      {open && (
        <div className="p-3 space-y-3">
          <div>
            <h4 className="text-xs font-bold mb-1">Query:</h4>
            <pre className="bg-gray-800 text-blue-300 text-xs p-2 rounded overflow-auto max-h-48 font-mono">
              {raw.query}
            </pre>
          </div>
          {raw.variables && (
            <div>
              <h4 className="text-xs font-bold mb-1">Variables:</h4>
              <JsonViewer data={raw.variables} maxHeight="150px" />
            </div>
          )}
          <div>
            <h4 className="text-xs font-bold mb-1">Response:</h4>
            <JsonViewer data={raw.response} maxHeight="300px" />
          </div>
          <button
            onClick={() => {
              const curl = `curl -X POST ${window.location.origin}/query \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer <TOKEN>" \\
  -d '${JSON.stringify({ query: raw.query, variables: raw.variables })}'`
              navigator.clipboard.writeText(curl)
            }}
            className="text-xs bg-gray-700 text-white px-2 py-1 rounded hover:bg-gray-600"
          >
            Copy as cURL
          </button>
        </div>
      )}
    </div>
  )
}
```

**Step 4: Commit**

```bash
git add frontend/src/hooks/ frontend/src/components/
git commit -m "feat(frontend): add useGraphQL hook, RawPanel, JsonViewer"
```

---

### Task 4: Layout + Navigation

**Files:**
- Create: `frontend/src/components/Layout.tsx`
- Modify: `frontend/src/App.tsx`

**Step 1: Create Layout with sidebar**

```tsx
// frontend/src/components/Layout.tsx
import { NavLink, Outlet } from 'react-router-dom'
import { useAuth } from '../auth/useAuth'

const navItems = [
  { to: '/catalog', label: 'Catalog' },
  { to: '/dictionary', label: 'Dictionary' },
  { to: '/study', label: 'Study' },
  { to: '/topics', label: 'Topics' },
  { to: '/inbox', label: 'Inbox' },
  { to: '/profile', label: 'Profile' },
  { to: '/explorer', label: 'API Explorer' },
]

export function Layout() {
  const { isAuthenticated, logout, token } = useAuth()

  return (
    <div className="flex h-screen">
      {/* Sidebar */}
      <nav className="w-56 bg-gray-900 text-gray-100 flex flex-col">
        <div className="p-4 text-lg font-bold border-b border-gray-700">
          MyEnglish Test
        </div>
        <div className="flex-1 overflow-y-auto py-2">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              className={({ isActive }) =>
                `block px-4 py-2 text-sm hover:bg-gray-800 ${isActive ? 'bg-gray-800 text-white font-medium' : 'text-gray-400'}`
              }
            >
              {item.label}
            </NavLink>
          ))}
        </div>
        <div className="p-4 border-t border-gray-700 text-xs">
          {isAuthenticated ? (
            <div className="space-y-2">
              <div className="text-green-400">Authenticated</div>
              <div className="truncate text-gray-500" title={token ?? ''}>
                {token?.slice(0, 20)}...
              </div>
              <button onClick={logout} className="text-red-400 hover:text-red-300">
                Logout
              </button>
            </div>
          ) : (
            <NavLink to="/login" className="text-yellow-400 hover:text-yellow-300">
              Login
            </NavLink>
          )}
        </div>
      </nav>

      {/* Main content */}
      <main className="flex-1 overflow-y-auto bg-gray-50">
        <Outlet />
      </main>
    </div>
  )
}
```

**Step 2: Update App.tsx with router structure**

```tsx
// frontend/src/App.tsx
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider } from './auth/AuthProvider'
import { Layout } from './components/Layout'

// Placeholder pages (will be replaced in subsequent tasks)
function Placeholder({ name }: { name: string }) {
  return <div className="p-6 text-gray-500">Page: {name} (coming soon)</div>
}

function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Placeholder name="Login" />} />
          <Route element={<Layout />}>
            <Route path="/" element={<Navigate to="/dictionary" replace />} />
            <Route path="/catalog" element={<Placeholder name="Catalog" />} />
            <Route path="/dictionary" element={<Placeholder name="Dictionary" />} />
            <Route path="/entry/:id" element={<Placeholder name="Entry Detail" />} />
            <Route path="/study" element={<Placeholder name="Study" />} />
            <Route path="/topics" element={<Placeholder name="Topics" />} />
            <Route path="/inbox" element={<Placeholder name="Inbox" />} />
            <Route path="/profile" element={<Placeholder name="Profile" />} />
            <Route path="/explorer" element={<Placeholder name="API Explorer" />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  )
}

export default App
```

**Step 3: Verify navigation works**

```bash
cd frontend && npm run dev
```

Expected: Sidebar visible, nav links highlight on click, pages show placeholder text.

**Step 4: Commit**

```bash
git add frontend/src/
git commit -m "feat(frontend): add Layout with sidebar navigation and routing"
```

---

### Task 5: Login Page

**Files:**
- Create: `frontend/src/auth/LoginPage.tsx`
- Modify: `frontend/src/App.tsx` (replace login placeholder)

**Step 1: Create LoginPage**

```tsx
// frontend/src/auth/LoginPage.tsx
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from './useAuth'

export function LoginPage() {
  const { setToken, isAuthenticated } = useAuth()
  const navigate = useNavigate()
  const [jwt, setJwt] = useState('')

  if (isAuthenticated) {
    navigate('/dictionary')
    return null
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="bg-white shadow-lg rounded-lg p-8 max-w-md w-full space-y-6">
        <h1 className="text-2xl font-bold text-center">MyEnglish Test Frontend</h1>

        {/* Manual JWT mode */}
        <div className="space-y-3">
          <h2 className="text-sm font-medium text-gray-700">Paste JWT Token</h2>
          <textarea
            value={jwt}
            onChange={(e) => setJwt(e.target.value)}
            placeholder="eyJhbGciOiJIUzI1NiIs..."
            className="w-full border rounded p-2 text-xs font-mono h-24 resize-none"
          />
          <button
            onClick={() => {
              const trimmed = jwt.trim()
              if (trimmed) {
                setToken(trimmed)
                navigate('/dictionary')
              }
            }}
            disabled={!jwt.trim()}
            className="w-full bg-blue-600 text-white py-2 rounded hover:bg-blue-700 disabled:opacity-50"
          >
            Use Token
          </button>
        </div>

        <div className="relative">
          <div className="absolute inset-0 flex items-center">
            <div className="w-full border-t border-gray-300" />
          </div>
          <div className="relative flex justify-center text-sm">
            <span className="px-2 bg-white text-gray-500">or</span>
          </div>
        </div>

        {/* Google OAuth (placeholder) */}
        <div className="space-y-3">
          <h2 className="text-sm font-medium text-gray-700">Google OAuth</h2>
          <p className="text-xs text-gray-500">
            Backend auth endpoints (login/refresh/logout) are not yet exposed via HTTP.
            Use manual JWT token for now. When REST auth endpoints are added, Google OAuth will work here.
          </p>
          <button
            disabled
            className="w-full bg-gray-200 text-gray-400 py-2 rounded cursor-not-allowed"
          >
            Login with Google (not yet available)
          </button>
        </div>

        <div className="text-xs text-gray-400 text-center">
          <p>To get a JWT: run the backend, create a user via tests or direct DB insert,
          then use the auth service to generate a token.</p>
        </div>
      </div>
    </div>
  )
}
```

**Step 2: Wire into App.tsx**

Replace login placeholder import with `LoginPage` component.

**Step 3: Verify**

Expected: Login page shows JWT paste field, Google OAuth shows as disabled with explanation.

**Step 4: Commit**

```bash
git add frontend/src/auth/LoginPage.tsx frontend/src/App.tsx
git commit -m "feat(frontend): add login page with JWT paste support"
```

---

### Task 6: Catalog Page (searchCatalog + previewRefEntry)

**Files:**
- Create: `frontend/src/pages/CatalogPage.tsx`
- Modify: `frontend/src/App.tsx` (replace placeholder)

**Step 1: Create CatalogPage**

The page has:
- Search input + button calling `searchCatalog(query, limit)`
- Results list showing RefEntry text + senses
- Click on entry calls `previewRefEntry(text)` to show full tree
- "Add to Dictionary" button (navigates to dictionary create flow)

Key GraphQL queries:
```graphql
query SearchCatalog($query: String!, $limit: Int) {
  searchCatalog(query: $query, limit: $limit) {
    id text textNormalized
    senses { id definition partOfSpeech cefrLevel position
      translations { id text }
      examples { id sentence translation }
    }
    pronunciations { id transcription audioUrl region }
    images { id url caption }
  }
}

query PreviewRefEntry($text: String!) {
  previewRefEntry(text: $text) {
    id text textNormalized
    senses { id definition partOfSpeech cefrLevel position
      translations { id text }
      examples { id sentence translation }
    }
    pronunciations { id transcription audioUrl region }
    images { id url caption }
  }
}
```

The page should:
1. Show search input with limit dropdown (5/10/20)
2. Display results as cards with senses collapsed/expandable
3. Have a "Preview" button per result for full fetch
4. Have "Add to Dictionary" button (createEntryFromCatalog with sense checkboxes)
5. Include RawPanel at bottom

**Step 2: Commit**

```bash
git add frontend/src/pages/CatalogPage.tsx frontend/src/App.tsx
git commit -m "feat(frontend): add Catalog page with search and preview"
```

---

### Task 7: Dictionary Page (list + filters + actions)

**Files:**
- Create: `frontend/src/pages/DictionaryPage.tsx`
- Modify: `frontend/src/App.tsx`

**Step 1: Create DictionaryPage**

The page has two tabs: "Active" and "Deleted".

**Active tab:**
- Filter bar: search input, hasCard toggle, partOfSpeech dropdown, topicId dropdown, status dropdown, sort field + direction
- Entry list table: text, notes (truncated), card status badge, senses count, created date
- Pagination: "first/after" (cursor) or "limit/offset" mode toggle
- Actions bar: "Create Custom", "Create from Catalog" (link to /catalog), "Import", "Export", "Batch Delete"
- Click on entry row → navigate to `/entry/:id`

**Deleted tab:**
- Simple list from `deletedEntries(limit, offset)`
- "Restore" button per entry
- Pagination via offset

Key GraphQL queries:
```graphql
query Dictionary($input: DictionaryFilterInput!) {
  dictionary(input: $input) {
    edges {
      cursor
      node {
        id text textNormalized notes createdAt updatedAt deletedAt
        senses { id definition partOfSpeech }
        card { id status nextReviewAt }
        topics { id name }
      }
    }
    pageInfo { hasNextPage hasPreviousPage startCursor endCursor }
    totalCount
  }
}

query DeletedEntries($limit: Int, $offset: Int) {
  deletedEntries(limit: $limit, offset: $offset) {
    entries { id text notes deletedAt }
    totalCount
  }
}

query ExportEntries {
  exportEntries {
    exportedAt
    items { text notes cardStatus createdAt
      senses { definition partOfSpeech translations examples { sentence translation } }
    }
  }
}
```

Key mutations:
```graphql
mutation CreateCustom($input: CreateEntryCustomInput!) {
  createEntryCustom(input: $input) { entry { id text } }
}

mutation DeleteEntry($id: UUID!) {
  deleteEntry(id: $id) { entryId }
}

mutation RestoreEntry($id: UUID!) {
  restoreEntry(id: $id) { entry { id text } }
}

mutation BatchDelete($ids: [UUID!]!) {
  batchDeleteEntries(ids: $ids) { deletedCount errors { id message } }
}

mutation Import($input: ImportEntriesInput!) {
  importEntries(input: $input) { importedCount skippedCount errors { index text message } }
}
```

**Important UI elements:**
- "Create Custom" opens a modal/inline form: text, senses (dynamic list), notes, createCard checkbox, topicId dropdown
- "Import" opens a textarea where user can paste JSON array of `[{text, translations, notes, topicName}]`
- "Export" button triggers query and shows downloadable JSON
- Batch delete: checkboxes per row, "Delete Selected" button
- Each entry row is clickable → navigate to `/entry/:id`

**Step 2: Commit**

```bash
git add frontend/src/pages/DictionaryPage.tsx frontend/src/App.tsx
git commit -m "feat(frontend): add Dictionary page with filters, CRUD, import/export"
```

---

### Task 8: Entry Detail Page (content editing)

**Files:**
- Create: `frontend/src/pages/EntryDetailPage.tsx`
- Modify: `frontend/src/App.tsx`

**Step 1: Create EntryDetailPage**

This is the most complex page. It shows:
1. **Entry header**: text, notes (editable), card status, topics list
2. **Senses section**: ordered list, each expandable
   - Definition, PartOfSpeech, CEFR level
   - Translations (ordered list, add/edit/delete/reorder)
   - Examples (ordered list, add/edit/delete/reorder)
3. **User Images section**: add (url + caption), delete
4. **Pronunciations**: read-only display (from catalog)
5. **Catalog Images**: read-only display (from catalog)
6. **Card section**: create card / delete card / card info (status, nextReview, interval, ease)

Key query:
```graphql
query DictionaryEntry($id: UUID!) {
  dictionaryEntry(id: $id) {
    id text textNormalized notes createdAt updatedAt
    senses {
      id definition partOfSpeech cefrLevel sourceSlug position
      translations { id text sourceSlug position }
      examples { id sentence translation sourceSlug position }
    }
    pronunciations { id transcription audioUrl region }
    catalogImages { id url caption }
    userImages { id url caption createdAt }
    card { id status nextReviewAt intervalDays easeFactor createdAt updatedAt }
    topics { id name }
  }
}
```

Content mutations (all with RawPanel):
```graphql
# Notes
mutation UpdateNotes($input: UpdateEntryNotesInput!) {
  updateEntryNotes(input: $input) { entry { id notes } }
}

# Senses
mutation AddSense($input: AddSenseInput!) {
  addSense(input: $input) { sense { id definition partOfSpeech cefrLevel position } }
}
mutation UpdateSense($input: UpdateSenseInput!) {
  updateSense(input: $input) { sense { id definition partOfSpeech cefrLevel } }
}
mutation DeleteSense($id: UUID!) { deleteSense(id: $id) { senseId } }
mutation ReorderSenses($input: ReorderSensesInput!) {
  reorderSenses(input: $input) { success }
}

# Translations
mutation AddTranslation($input: AddTranslationInput!) {
  addTranslation(input: $input) { translation { id text position } }
}
mutation UpdateTranslation($input: UpdateTranslationInput!) {
  updateTranslation(input: $input) { translation { id text } }
}
mutation DeleteTranslation($id: UUID!) { deleteTranslation(id: $id) { translationId } }
mutation ReorderTranslations($input: ReorderTranslationsInput!) {
  reorderTranslations(input: $input) { success }
}

# Examples
mutation AddExample($input: AddExampleInput!) {
  addExample(input: $input) { example { id sentence translation position } }
}
mutation UpdateExample($input: UpdateExampleInput!) {
  updateExample(input: $input) { example { id sentence translation } }
}
mutation DeleteExample($id: UUID!) { deleteExample(id: $id) { exampleId } }
mutation ReorderExamples($input: ReorderExamplesInput!) {
  reorderExamples(input: $input) { success }
}

# User Images
mutation AddUserImage($input: AddUserImageInput!) {
  addUserImage(input: $input) { image { id url caption createdAt } }
}
mutation DeleteUserImage($id: UUID!) { deleteUserImage(id: $id) { imageId } }

# Card
mutation CreateCard($entryId: UUID!) {
  createCard(entryId: $entryId) { card { id status } }
}
mutation DeleteCard($id: UUID!) { deleteCard(id: $id) { cardId } }
```

**UI approach:**
- Each sense is a collapsible card with inline edit buttons
- Translations and examples show as lists with +/edit/delete/drag-to-reorder buttons
- Reorder uses simple "move up / move down" buttons (sends `reorder*` mutation with new positions)
- After any mutation, refetch the full entry to show updated state
- RawPanel at the bottom shows the last request

**Step 2: Commit**

```bash
git add frontend/src/pages/EntryDetailPage.tsx frontend/src/App.tsx
git commit -m "feat(frontend): add Entry Detail page with content editing"
```

---

### Task 9: Study Page (dashboard + review + sessions)

**Files:**
- Create: `frontend/src/pages/StudyPage.tsx`
- Modify: `frontend/src/App.tsx`

**Step 1: Create StudyPage**

Sections:
1. **Dashboard** (top): dueCount, newCount, reviewedToday, streak, statusCounts, overdueCount, activeSession info
2. **Session controls**: Start Session / Finish Session / Abandon Session
3. **Study Queue**: list of entries due for review (with limit selector)
4. **Review Mode**: click "Start Review" → show one card at a time → 4 grade buttons
5. **Card Actions**: batch create cards (select entries without cards)
6. **Card Inspector**: select a card → view history + stats, undo last review

Key queries:
```graphql
query Dashboard {
  dashboard {
    dueCount newCount reviewedToday streak overdueCount
    statusCounts { new learning review mastered }
    activeSession { id status startedAt finishedAt result { totalReviews gradeCounts { again hard good easy } averageDurationMs } }
  }
}

query StudyQueue($limit: Int) {
  studyQueue(limit: $limit) {
    id text
    senses { id definition partOfSpeech translations { id text } }
    card { id status nextReviewAt intervalDays easeFactor }
  }
}

query CardHistory($input: GetCardHistoryInput!) {
  cardHistory(input: $input) { id cardId grade durationMs reviewedAt }
}

query CardStats($cardId: UUID!) {
  cardStats(cardId: $cardId) { totalReviews averageDurationMs accuracy gradeDistribution { again hard good easy } }
}
```

Key mutations:
```graphql
mutation ReviewCard($input: ReviewCardInput!) {
  reviewCard(input: $input) { card { id status nextReviewAt intervalDays easeFactor } }
}

mutation UndoReview($cardId: UUID!) {
  undoReview(cardId: $cardId) { card { id status nextReviewAt intervalDays easeFactor } }
}

mutation BatchCreateCards($entryIds: [UUID!]!) {
  batchCreateCards(entryIds: $entryIds) { createdCount skippedCount errors { entryId message } }
}

mutation StartSession { startStudySession { session { id status startedAt } } }

mutation FinishSession($input: FinishSessionInput!) {
  finishStudySession(input: $input) { session { id status finishedAt result { totalReviews gradeCounts { again hard good easy } averageDurationMs } } }
}

mutation AbandonSession { abandonStudySession { success } }
```

**Review flow UI:**
1. Click "Start Review" → calls `startStudySession` + fetches `studyQueue`
2. Shows first entry: text, definition, translations (initially hidden, click to reveal)
3. After reveal: 4 buttons (Again / Hard / Good / Easy) → calls `reviewCard` with `durationMs` (time from card shown to grade click)
4. After grade: shows updated card state briefly, moves to next card
5. "Undo" button: calls `undoReview`, re-shows previous card
6. When queue exhausted: calls `finishStudySession`, shows results

**Step 2: Commit**

```bash
git add frontend/src/pages/StudyPage.tsx frontend/src/App.tsx
git commit -m "feat(frontend): add Study page with dashboard, review flow, sessions"
```

---

### Task 10: Topics Page

**Files:**
- Create: `frontend/src/pages/TopicsPage.tsx`
- Modify: `frontend/src/App.tsx`

**Step 1: Create TopicsPage**

- List of topics as cards: name, description, entryCount
- "Create Topic" form: name + description
- Edit topic inline: click name to edit, click description to edit
- Delete topic button with confirmation
- Expand topic → shows entries linked, with "Unlink" button
- "Link Entry" form: entryId input (or dropdown from dictionary)
- "Batch Link" form: paste entry IDs

Key queries/mutations:
```graphql
query Topics {
  topics { id name description entryCount createdAt updatedAt }
}

mutation CreateTopic($input: CreateTopicInput!) {
  createTopic(input: $input) { topic { id name description entryCount } }
}

mutation UpdateTopic($input: UpdateTopicInput!) {
  updateTopic(input: $input) { topic { id name description } }
}

mutation DeleteTopic($id: UUID!) { deleteTopic(id: $id) { topicId } }

mutation LinkEntry($input: LinkEntryInput!) {
  linkEntryToTopic(input: $input) { success }
}

mutation UnlinkEntry($input: UnlinkEntryInput!) {
  unlinkEntryFromTopic(input: $input) { success }
}

mutation BatchLink($input: BatchLinkEntriesInput!) {
  batchLinkEntriesToTopic(input: $input) { linked skipped }
}
```

**Step 2: Commit**

```bash
git add frontend/src/pages/TopicsPage.tsx frontend/src/App.tsx
git commit -m "feat(frontend): add Topics page with CRUD and entry linking"
```

---

### Task 11: Inbox Page

**Files:**
- Create: `frontend/src/pages/InboxPage.tsx`
- Modify: `frontend/src/App.tsx`

**Step 1: Create InboxPage**

Simplest page:
- Create form: text (required) + context (optional)
- List with pagination (limit/offset)
- Each item: text, context, createdAt, delete button
- "Clear All" button with confirmation
- Total count display

Key queries/mutations:
```graphql
query InboxItems($limit: Int, $offset: Int) {
  inboxItems(limit: $limit, offset: $offset) {
    items { id text context createdAt }
    totalCount
  }
}

query InboxItem($id: UUID!) {
  inboxItem(id: $id) { id text context createdAt }
}

mutation CreateInboxItem($input: CreateInboxItemInput!) {
  createInboxItem(input: $input) { item { id text context createdAt } }
}

mutation DeleteInboxItem($id: UUID!) {
  deleteInboxItem(id: $id) { itemId }
}

mutation ClearInbox { clearInbox { deletedCount } }
```

**Step 2: Commit**

```bash
git add frontend/src/pages/InboxPage.tsx frontend/src/App.tsx
git commit -m "feat(frontend): add Inbox page with CRUD and clear all"
```

---

### Task 12: Profile Page (me + settings)

**Files:**
- Create: `frontend/src/pages/ProfilePage.tsx`
- Modify: `frontend/src/App.tsx`

**Step 1: Create ProfilePage**

Two sections:
1. **Profile**: display user info from `me` query (id, email, name, avatar, provider, createdAt, settings)
2. **Settings form**: newCardsPerDay (number), reviewsPerDay (number), maxIntervalDays (number), timezone (text) — save via `updateSettings`

Key queries/mutations:
```graphql
query Me {
  me {
    id email name avatarUrl oauthProvider createdAt
    settings { newCardsPerDay reviewsPerDay maxIntervalDays timezone }
  }
}

mutation UpdateSettings($input: UpdateSettingsInput!) {
  updateSettings(input: $input) {
    settings { newCardsPerDay reviewsPerDay maxIntervalDays timezone }
  }
}
```

**Step 2: Commit**

```bash
git add frontend/src/pages/ProfilePage.tsx frontend/src/App.tsx
git commit -m "feat(frontend): add Profile page with user info and SRS settings"
```

---

### Task 13: API Explorer Page

**Files:**
- Create: `frontend/src/pages/ExplorerPage.tsx`
- Create: `frontend/src/pages/explorer/ExplorerSection.tsx`
- Create: `frontend/src/pages/explorer/OperationForm.tsx`
- Create: `frontend/src/pages/explorer/operations.ts`

**Step 1: Create operations registry**

`operations.ts` contains a static list of ALL GraphQL operations organized by domain:

```typescript
interface OperationField {
  name: string
  type: 'string' | 'number' | 'boolean' | 'uuid' | 'json' | 'enum' | 'uuid[]'
  required?: boolean
  enumValues?: string[]
  placeholder?: string
}

interface Operation {
  name: string
  type: 'query' | 'mutation'
  query: string
  fields: OperationField[]
}

interface OperationGroup {
  name: string
  operations: Operation[]
}
```

Groups:
- **Dictionary**: searchCatalog, previewRefEntry, dictionary, dictionaryEntry, deletedEntries, exportEntries, createEntryFromCatalog, createEntryCustom, updateEntryNotes, deleteEntry, restoreEntry, batchDeleteEntries, importEntries
- **Content**: addSense, updateSense, deleteSense, reorderSenses, addTranslation, updateTranslation, deleteTranslation, reorderTranslations, addExample, updateExample, deleteExample, reorderExamples, addUserImage, deleteUserImage
- **Study**: studyQueue, dashboard, cardHistory, cardStats, reviewCard, undoReview, createCard, deleteCard, batchCreateCards, startStudySession, finishStudySession, abandonStudySession
- **Organization**: topics, inboxItems, inboxItem, createTopic, updateTopic, deleteTopic, linkEntryToTopic, unlinkEntryFromTopic, batchLinkEntriesToTopic, createInboxItem, deleteInboxItem, clearInbox
- **User**: me, updateSettings

Every operation includes the FULL GraphQL query string with all fields.

**Step 2: Create OperationForm**

A generic component that takes an `Operation` and renders:
- Form fields based on `Operation.fields` types
- "Execute" button
- Loading state
- JsonViewer for result
- Error display with field highlighting for VALIDATION errors

**Step 3: Create ExplorerSection**

Tab group that renders `OperationForm` for each operation in a group. Operations are collapsible — click to expand form.

**Step 4: Create ExplorerPage**

Top-level tabs (Dictionary | Content | Study | Organization | User), each renders ExplorerSection with the corresponding group.

**Step 5: Commit**

```bash
git add frontend/src/pages/ExplorerPage.tsx frontend/src/pages/explorer/
git commit -m "feat(frontend): add API Explorer with all operations"
```

---

### Task 14: Error Handling + Toast Notifications

**Files:**
- Create: `frontend/src/components/Toast.tsx`
- Create: `frontend/src/components/ToastProvider.tsx`
- Modify: `frontend/src/App.tsx`

**Step 1: Create Toast system**

Simple toast notification system:
- `ToastProvider` wraps the app, provides `addToast(message, type)` via context
- `useToast()` hook
- Types: success (green), error (red), warning (yellow), info (blue)
- Auto-dismiss after 5 seconds, with close button
- Display GraphQL error codes as toast on mutation failures

**Step 2: Wire into useGraphQL hook**

Add optional `onError` callback that triggers toast with error code and message.

**Step 3: Commit**

```bash
git add frontend/src/components/Toast.tsx frontend/src/components/ToastProvider.tsx
git commit -m "feat(frontend): add toast notification system for errors"
```

---

### Task 15: Polish + Shared Components

**Files:**
- Create: `frontend/src/components/StatusBadge.tsx`
- Create: `frontend/src/components/Pagination.tsx`
- Create: `frontend/src/components/ConfirmDialog.tsx`
- Create: `frontend/src/components/LoadingSpinner.tsx`

**Step 1: Create shared components**

- `StatusBadge`: colored badge for LearningStatus (NEW=blue, LEARNING=yellow, REVIEW=purple, MASTERED=green)
- `Pagination`: offset-based (prev/next + page number) and cursor-based (Load More) modes
- `ConfirmDialog`: simple modal for delete confirmations
- `LoadingSpinner`: spinning indicator for loading states

**Step 2: Integrate into existing pages**

Replace inline loading/pagination/status displays with shared components.

**Step 3: Final verification**

```bash
cd frontend && npm run build
```

Expected: Build succeeds with no TypeScript errors.

**Step 4: Commit**

```bash
git add frontend/src/components/
git commit -m "feat(frontend): add shared components (badges, pagination, dialogs)"
```

---

### Task 16: Final Wiring + README

**Files:**
- Modify: `frontend/src/App.tsx` (replace all remaining placeholders)
- Create: `frontend/README.md`

**Step 1: Ensure all page imports are wired**

Verify every route in App.tsx uses the real page component, not a placeholder.

**Step 2: Create frontend README**

Brief README with:
- How to run: `cd frontend && npm install && npm run dev`
- Backend URL configuration (proxy in vite.config.ts)
- Auth: paste JWT token on login page
- Pages overview
- CORS: backend must allow `http://localhost:3000`

**Step 3: Final build check**

```bash
cd frontend && npm run build && npm run preview
```

**Step 4: Commit**

```bash
git add frontend/
git commit -m "feat(frontend): complete test frontend with all pages wired"
```
