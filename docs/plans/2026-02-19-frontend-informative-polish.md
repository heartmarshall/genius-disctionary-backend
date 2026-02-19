# Frontend Informative Polish Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make the test frontend more informative by auto-loading data on page mount, adding summary stats to Dictionary and Study pages, improving empty states, and adding sidebar icons.

**Architecture:** Pure frontend changes to existing React/TypeScript pages. Each page gets a `useEffect` to auto-fetch data on mount when authenticated. Dictionary and Study pages get new summary sections computed from already-fetched data. Sidebar gets inline SVG icons.

**Tech Stack:** React 19, TypeScript, Tailwind CSS 4, Vite 7

---

### Task 1: Sidebar Icons and Branding

**Files:**
- Modify: `frontend/src/components/Layout.tsx`

**Step 1: Update the nav items array and sidebar rendering**

Replace the entire `Layout.tsx` file content. Add inline SVG icons to each nav item, change the header to "MyEnglish" with a "Test Environment" subtitle.

```tsx
import { NavLink, Outlet } from 'react-router-dom'
import { useAuth } from '../auth/useAuth'

const navItems = [
  {
    to: '/catalog',
    label: 'Catalog',
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" d="M12 6.042A8.967 8.967 0 0 0 6 3.75c-1.052 0-2.062.18-3 .512v14.25A8.987 8.987 0 0 1 6 18c2.305 0 4.408.867 6 2.292m0-14.25a8.966 8.966 0 0 1 6-2.292c1.052 0 2.062.18 3 .512v14.25A8.987 8.987 0 0 0 18 18a8.967 8.967 0 0 0-6 2.292m0-14.25v14.25" />
      </svg>
    ),
  },
  {
    to: '/dictionary',
    label: 'Dictionary',
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 12h16.5m-16.5 3.75h16.5M3.75 19.5h16.5M5.625 4.5h12.75a1.875 1.875 0 0 1 0 3.75H5.625a1.875 1.875 0 0 1 0-3.75Z" />
      </svg>
    ),
  },
  {
    to: '/study',
    label: 'Study',
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" d="M4.26 10.147a60.438 60.438 0 0 0-.491 6.347A48.62 48.62 0 0 1 12 20.904a48.62 48.62 0 0 1 8.232-4.41 60.46 60.46 0 0 0-.491-6.347m-15.482 0a50.636 50.636 0 0 0-2.658-.813A59.906 59.906 0 0 1 12 3.493a59.903 59.903 0 0 1 10.399 5.84c-.896.248-1.783.52-2.658.814m-15.482 0A50.717 50.717 0 0 1 12 13.489a50.702 50.702 0 0 1 7.74-3.342M6.75 15a.75.75 0 1 0 0-1.5.75.75 0 0 0 0 1.5Zm0 0v-3.675A55.378 55.378 0 0 1 12 8.443m-7.007 11.55A5.981 5.981 0 0 0 6.75 15.75v-1.5" />
      </svg>
    ),
  },
  {
    to: '/topics',
    label: 'Topics',
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" d="M9.568 3H5.25A2.25 2.25 0 0 0 3 5.25v4.318c0 .597.237 1.17.659 1.591l9.581 9.581c.699.699 1.78.872 2.607.33a18.095 18.095 0 0 0 5.223-5.223c.542-.827.369-1.908-.33-2.607L11.16 3.66A2.25 2.25 0 0 0 9.568 3Z" />
        <path strokeLinecap="round" strokeLinejoin="round" d="M6 6h.008v.008H6V6Z" />
      </svg>
    ),
  },
  {
    to: '/inbox',
    label: 'Inbox',
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 13.5h3.86a2.25 2.25 0 0 1 2.012 1.244l.256.512a2.25 2.25 0 0 0 2.013 1.244h3.218a2.25 2.25 0 0 0 2.013-1.244l.256-.512a2.25 2.25 0 0 1 2.013-1.244h3.859m-19.5.338V18a2.25 2.25 0 0 0 2.25 2.25h15A2.25 2.25 0 0 0 21.75 18v-4.162c0-.224-.034-.447-.1-.661L19.24 5.338a2.25 2.25 0 0 0-2.15-1.588H6.911a2.25 2.25 0 0 0-2.15 1.588L2.35 13.177a2.25 2.25 0 0 0-.1.661Z" />
      </svg>
    ),
  },
  {
    to: '/profile',
    label: 'Profile',
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 6a3.75 3.75 0 1 1-7.5 0 3.75 3.75 0 0 1 7.5 0ZM4.501 20.118a7.5 7.5 0 0 1 14.998 0A17.933 17.933 0 0 1 12 21.75c-2.676 0-5.216-.584-7.499-1.632Z" />
      </svg>
    ),
  },
  {
    to: '/explorer',
    label: 'API Explorer',
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" d="M17.25 6.75 22.5 12l-5.25 5.25m-10.5 0L1.5 12l5.25-5.25m7.5-3-4.5 16.5" />
      </svg>
    ),
  },
]

export function Layout() {
  const { isAuthenticated, logout, token } = useAuth()

  return (
    <div className="flex h-screen">
      <nav className="w-56 bg-gray-900 text-gray-100 flex flex-col shrink-0">
        <div className="p-4 border-b border-gray-700">
          <div className="text-lg font-bold">MyEnglish</div>
          <div className="text-xs text-gray-500">Test Environment</div>
        </div>
        <div className="flex-1 overflow-y-auto py-2">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              className={({ isActive }) =>
                `flex items-center gap-2.5 px-4 py-2 text-sm hover:bg-gray-800 ${isActive ? 'bg-gray-800 text-white font-medium' : 'text-gray-400'}`
              }
            >
              {item.icon}
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

      <main className="flex-1 overflow-y-auto bg-gray-50">
        <Outlet />
      </main>
    </div>
  )
}
```

**Step 2: Verify visually**

Run: `cd frontend && npm run dev`

Open `http://localhost:3000` and verify:
- Each sidebar nav item has an icon to the left of the label
- Header says "MyEnglish" with "Test Environment" subtitle
- Icons are the same color as the text (gray-400 inactive, white active)

**Step 3: Commit**

```bash
git add frontend/src/components/Layout.tsx
git commit -m "feat(frontend): add sidebar icons and update branding"
```

---

### Task 2: Dictionary Page - Auto-load and Summary Bar

**Files:**
- Modify: `frontend/src/pages/DictionaryPage.tsx`

**Step 1: Add auto-load on mount**

At the top of the `DictionaryPage` component function, after the existing hook declarations, add a `useEffect` that auto-fetches dictionary entries when the component mounts and the user is authenticated. Add `useEffect` to the existing `useState` import.

Find the import line:
```tsx
import { useState } from 'react'
```

Replace with:
```tsx
import { useState, useEffect } from 'react'
```

Then after the line:
```tsx
  const lastRaw = createCustom.raw ?? restoreEntry.raw
    ?? batchDelete.raw ?? importEntries.raw ?? exportEntries.raw
    ?? deletedEntries.raw ?? dictionary.raw
```

Add:
```tsx
  // Auto-load dictionary on mount
  useEffect(() => {
    if (isAuthenticated) {
      dictionary.execute(DICTIONARY_QUERY, { input: buildFilterInput() })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])
```

**Step 2: Add summary bar component**

Before the `renderFilterBar()` function, add a new `renderSummaryBar()` function:

```tsx
  function renderSummaryBar() {
    if (!dictionary.data) return null
    const edges = dictionary.data.dictionary.edges
    const totalCount = dictionary.data.dictionary.totalCount

    // Compute stats from loaded entries
    let withCard = 0
    let withoutCard = 0
    const statusCounts: Record<string, number> = { NEW: 0, LEARNING: 0, REVIEW: 0, MASTERED: 0 }

    for (const edge of edges) {
      if (edge.node.card) {
        withCard++
        const status = edge.node.card.status
        if (status in statusCounts) statusCounts[status]++
      } else {
        withoutCard++
      }
    }

    const statusColors: Record<string, string> = {
      NEW: 'bg-blue-50 border-blue-200 text-blue-700',
      LEARNING: 'bg-yellow-50 border-yellow-200 text-yellow-700',
      REVIEW: 'bg-purple-50 border-purple-200 text-purple-700',
      MASTERED: 'bg-green-50 border-green-200 text-green-700',
    }

    return (
      <div className="grid grid-cols-3 md:grid-cols-7 gap-3">
        <div className="bg-white border border-gray-200 rounded-lg p-3 text-center">
          <div className="text-2xl font-bold text-gray-800">{totalCount}</div>
          <div className="text-xs text-gray-500">Total Entries</div>
        </div>
        <div className="bg-white border border-gray-200 rounded-lg p-3 text-center">
          <div className="text-2xl font-bold text-gray-700">{withCard}</div>
          <div className="text-xs text-gray-500">With Card</div>
        </div>
        <div className="bg-white border border-gray-200 rounded-lg p-3 text-center">
          <div className="text-2xl font-bold text-gray-400">{withoutCard}</div>
          <div className="text-xs text-gray-500">No Card</div>
        </div>
        {Object.entries(statusCounts).map(([status, count]) => (
          <div key={status} className={`border rounded-lg p-3 text-center ${statusColors[status]}`}>
            <div className="text-2xl font-bold">{count}</div>
            <div className="text-xs">{status}</div>
          </div>
        ))}
      </div>
    )
  }
```

**Step 3: Insert summary bar into render and improve empty state**

In the main render section, inside `{activeTab === 'active' && (`, add `{renderSummaryBar()}` as the first child before `{renderFilterBar()}`:

Find:
```tsx
      {activeTab === 'active' && (
        <div className="space-y-4">
          {renderFilterBar()}
```

Replace with:
```tsx
      {activeTab === 'active' && (
        <div className="space-y-4">
          {renderSummaryBar()}
          {renderFilterBar()}
```

Then in `renderEntryTable()`, update the empty state message. Find:
```tsx
      return <div className="text-gray-500 text-sm">No entries found.</div>
```

Replace with:
```tsx
      return (
        <div className="text-center py-8">
          <div className="text-gray-400 text-sm">No entries found.</div>
          <div className="text-gray-400 text-xs mt-1">
            Visit the Catalog to search and add your first words, or adjust your filters.
          </div>
        </div>
      )
```

**Step 4: Upgrade table header background**

In `renderEntryTable()`, find:
```tsx
          <thead className="bg-gray-50">
```

Replace with:
```tsx
          <thead className="bg-gray-100">
```

**Step 5: Verify visually**

Run dev server, navigate to `/dictionary`. Verify:
- Data auto-loads on page visit (no need to click "Apply Filters")
- Summary bar shows above filters with Total, With Card, No Card, and 4 status counts
- Empty state has a helpful message
- Table header is slightly more distinct

**Step 6: Commit**

```bash
git add frontend/src/pages/DictionaryPage.tsx
git commit -m "feat(frontend): auto-load dictionary, add summary bar and better empty state"
```

---

### Task 3: Study Page - Auto-load Dashboard and Queue

**Files:**
- Modify: `frontend/src/pages/StudyPage.tsx`

**Step 1: Add auto-load imports and effects**

Add `useEffect` to the import. Find:
```tsx
import { useState, useRef, useCallback } from 'react'
```

Replace with:
```tsx
import { useState, useRef, useCallback, useEffect } from 'react'
```

Then we need access to `isAuthenticated`. Add auth import after the existing imports:
```tsx
import { useAuth } from '../auth/useAuth'
```

At the start of the `StudyPage` component function, add:
```tsx
  const { isAuthenticated } = useAuth()
```

After the `lastRaw` declaration, add auto-load effects:
```tsx
  // Auto-load dashboard on mount
  useEffect(() => {
    if (isAuthenticated) {
      dashboard.execute(DASHBOARD_QUERY)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // Auto-load study queue after dashboard loads
  useEffect(() => {
    if (isAuthenticated && dashboard.data && !studyQueue.data && !studyQueue.loading) {
      studyQueue.execute(STUDY_QUEUE_QUERY, { limit: queueLimit }).then((data) => {
        if (data?.studyQueue) {
          setQueue(data.studyQueue)
        }
      })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dashboard.data])
```

**Step 2: Replace "Load Dashboard" button with refresh button**

In `renderDashboard()`, find:
```tsx
          <button
            onClick={handleLoadDashboard}
            disabled={dashboard.loading}
            className="px-4 py-1.5 text-sm bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
          >
            {dashboard.loading ? 'Loading...' : 'Load Dashboard'}
          </button>
```

Replace with:
```tsx
          <button
            onClick={handleLoadDashboard}
            disabled={dashboard.loading}
            className="px-2 py-1.5 text-sm text-gray-500 hover:text-gray-700 disabled:opacity-50"
            title="Refresh dashboard"
          >
            {dashboard.loading ? (
              <span className="text-xs">Loading...</span>
            ) : (
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0 3.181 3.183a8.25 8.25 0 0 0 13.803-3.7M4.031 9.865a8.25 8.25 0 0 1 13.803-3.7l3.181 3.182" />
              </svg>
            )}
          </button>
```

**Step 3: Add loading state for dashboard initial load**

In `renderDashboard()`, after the errors block and before the `{d && (` block, add an initial loading indicator. Find:
```tsx
        {dashboard.errors && (
          <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
            {dashboard.errors.map((err, i) => <div key={i}>{err.message}</div>)}
          </div>
        )}

        {d && (
```

Replace with:
```tsx
        {dashboard.errors && (
          <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
            {dashboard.errors.map((err, i) => <div key={i}>{err.message}</div>)}
          </div>
        )}

        {!d && dashboard.loading && (
          <div className="text-center py-6 text-gray-400 text-sm">Loading dashboard...</div>
        )}

        {!d && !dashboard.loading && !dashboard.errors && (
          <div className="text-center py-6 text-gray-400 text-sm">
            Log in to see your study dashboard.
          </div>
        )}

        {d && (
```

**Step 4: Verify visually**

Run dev server, navigate to `/study`. Verify:
- Dashboard auto-loads on page visit
- Study queue auto-loads after dashboard
- Refresh button (icon) replaces "Load Dashboard" button
- When not authenticated, shows helpful message

**Step 5: Commit**

```bash
git add frontend/src/pages/StudyPage.tsx
git commit -m "feat(frontend): auto-load study dashboard and queue on mount"
```

---

### Task 4: Topics Page - Auto-load

**Files:**
- Modify: `frontend/src/pages/TopicsPage.tsx`

**Step 1: Add auto-load**

Add `useEffect` to the import. Find:
```tsx
import { useState } from 'react'
```

Replace with:
```tsx
import { useState, useEffect } from 'react'
```

Add auth import:
```tsx
import { useAuth } from '../auth/useAuth'
```

At the start of the `TopicsPage` component function, add:
```tsx
  const { isAuthenticated } = useAuth()
```

After the `lastRaw` declaration, add:
```tsx
  // Auto-load topics on mount
  useEffect(() => {
    if (isAuthenticated) {
      topicsQuery.execute(TOPICS_QUERY)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])
```

**Step 2: Improve empty state**

Find:
```tsx
            <div className="text-gray-500 text-sm">No topics found.</div>
```

Replace with:
```tsx
            <div className="text-center py-6">
              <div className="text-gray-400 text-sm">No topics yet.</div>
              <div className="text-gray-400 text-xs mt-1">Create your first topic above to organize your vocabulary.</div>
            </div>
```

**Step 3: Verify visually**

Navigate to `/topics`. Topics list auto-loads. Empty state shows helpful message.

**Step 4: Commit**

```bash
git add frontend/src/pages/TopicsPage.tsx
git commit -m "feat(frontend): auto-load topics and improve empty state"
```

---

### Task 5: Inbox Page - Auto-load

**Files:**
- Modify: `frontend/src/pages/InboxPage.tsx`

**Step 1: Add auto-load**

Add `useEffect` to the import. Find:
```tsx
import { useState } from 'react'
```

Replace with:
```tsx
import { useState, useEffect } from 'react'
```

Add auth import:
```tsx
import { useAuth } from '../auth/useAuth'
```

At the start of the `InboxPage` component function, add:
```tsx
  const { isAuthenticated } = useAuth()
```

After the `lastRaw` declaration, add:
```tsx
  // Auto-load inbox items on mount
  useEffect(() => {
    if (isAuthenticated) {
      listItems.execute(INBOX_ITEMS_QUERY, { limit, offset })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])
```

**Step 2: Improve empty state**

Find:
```tsx
            <div className="text-gray-500 text-sm">No inbox items.</div>
```

Replace with:
```tsx
            <div className="text-center py-6">
              <div className="text-gray-400 text-sm">Your inbox is empty.</div>
              <div className="text-gray-400 text-xs mt-1">Add words you encounter for later review using the form above.</div>
            </div>
```

**Step 3: Verify visually**

Navigate to `/inbox`. Items auto-load. Empty state shows helpful guidance.

**Step 4: Commit**

```bash
git add frontend/src/pages/InboxPage.tsx
git commit -m "feat(frontend): auto-load inbox and improve empty state"
```

---

### Task 6: Final Visual Polish Pass

**Files:**
- Modify: `frontend/src/pages/StudyPage.tsx` (table header)
- Modify: `frontend/src/pages/DictionaryPage.tsx` (page subtitle)

**Step 1: Study page - upgrade card history table header**

In `StudyPage.tsx`, in `renderCardInspector()`, find:
```tsx
                <table className="w-full text-sm border border-gray-200 rounded">
                  <thead className="bg-gray-50">
```

Replace with:
```tsx
                <table className="w-full text-sm border border-gray-200 rounded">
                  <thead className="bg-gray-100">
```

**Step 2: Dictionary page - improve page subtitle**

In `DictionaryPage.tsx`, find:
```tsx
      <p className="text-sm text-gray-500">
        Manage your vocabulary entries. All operations require authentication.
      </p>
```

Replace with:
```tsx
      <p className="text-sm text-gray-500">
        Manage your vocabulary entries. Data loads automatically when authenticated.
      </p>
```

**Step 3: Study page - improve page subtitle**

In `StudyPage.tsx`, find:
```tsx
      <p className="text-sm text-gray-500">
        Spaced repetition study flow. Load dashboard, start sessions, review cards, inspect history.
      </p>
```

Replace with:
```tsx
      <p className="text-sm text-gray-500">
        Spaced repetition study flow. Dashboard and queue load automatically.
      </p>
```

**Step 4: Verify all pages**

Navigate through all pages. Verify:
- Sidebar has icons on all nav items
- Dictionary auto-loads with summary bar
- Study auto-loads dashboard and queue
- Topics auto-loads
- Inbox auto-loads
- All empty states have helpful messages
- Table headers are slightly more distinct
- Page subtitles reflect auto-loading behavior

**Step 5: Commit**

```bash
git add frontend/src/pages/StudyPage.tsx frontend/src/pages/DictionaryPage.tsx
git commit -m "feat(frontend): final visual polish - table headers and subtitles"
```
