# Phase 5: Organization (Topics + Inbox) — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Tools for organizing the dictionary — topics for grouping words by theme, and inbox for quick capture of unfamiliar words. Two independent features sharing one phase because both are relatively small.

**Architecture:** Both features are fully GraphQL-based. Topics have a many-to-many relationship with entries (an entry can belong to multiple topics). Inbox is a simple flat list of notes, completely disconnected from entries/cards — just a scratchpad. Both use straightforward CRUD patterns.

**Key references:**
- API: `backend_v4/docs/API.md` — Organization section (lines 166–182)
- Domain model: `docs/business/DOMAIN_MODEL.md` — Topic (many-to-many with Entry), Inbox Item
- Business rules: `docs/business/BUSINESS_RULES.md` — topic name max 100 chars, description max 500 chars, batch link max 200 entries, inbox text 1-500 chars, context max 2000 chars, inbox pagination max 200
- Workflows: `docs/business/WORKFLOWS.md` — "Organizing with Topics", "Inbox (Quick Capture)"
- Open questions: `docs/business/UNKNOWNS.md` — "Intended relationship between Inbox and Dictionary?" (inbox items are disconnected from entries)
- Design: `frontend-real/design-docs/palette-v3.html` — Buttons, Input Fields, Modals, Tags

---

## Task 1: Topics GraphQL Layer

**Goal:** All queries and mutations for topics, typed.

**What to do:**
- Create `src/graphql/queries/topics.ts`:
  - `TOPICS_QUERY` — list all topics with entry counts
- Create `src/graphql/mutations/topics.ts`:
  - `CREATE_TOPIC`, `UPDATE_TOPIC`, `DELETE_TOPIC`
  - `LINK_ENTRY_TO_TOPIC`, `BATCH_LINK_ENTRIES_TO_TOPIC`
- Define types in `src/types/topics.ts`:
  - `Topic`: `{ id, name, description, entryCount }`

**Commit:** `feat(topics): add GraphQL queries, mutations, and types`

---

## Task 2: Topics Page

**Goal:** Page to view, create, edit, and delete topics.

**What to do:**
- Replace stub `src/pages/TopicsPage.tsx`:
  - **Topics list**: cards or rows showing each topic — name, description (truncated), entry count badge
  - **Create topic**: "New Topic" button → inline form or dialog with name (required, max 100) + description (optional, max 500)
  - **Edit topic**: click topic name or edit icon → inline edit or dialog for name + description
  - **Delete topic**: delete button with confirmation dialog. Deleting a topic does NOT delete entries — just unlinks them.
  - **Click topic**: navigate to dictionary filtered by this topic (use existing DictionaryPage with `topicId` filter from Phase 3). Route: `/topics/:id` → redirect to `/dictionary?topicId=:id` or show entries inline.
- Create `src/hooks/useTopics.ts` — wraps `useQuery(TOPICS_QUERY)`, returns `{ topics, loading, error }`
- Create `src/components/topics/TopicCard.tsx` — single topic display with name, description, count, edit/delete actions

**Commit:** `feat(topics): implement topics page with CRUD`

---

## Task 3: Link Entries to Topics

**Goal:** User can assign entries to topics from both directions — from topic page and from entry detail page.

**Context:** `linkEntryToTopic` links one entry, `batchLinkEntriesToTopic` links up to 200 at once. Entry detail page (Phase 3) already shows linked topics — now add the ability to link/unlink.

**What to do:**
- **From Entry Detail Page** (modify `src/pages/DictionaryEntryPage.tsx`):
  - Topics section already shows linked topics as tags
  - Add "Add to Topic" button → dropdown/dialog listing all topics with checkboxes
  - Toggle: link or unlink entry from topic
  - Call `linkEntryToTopic` for linking (unlink mutation — check if API supports it, otherwise this is a gap to note)
- **From Topics Page** — batch linking:
  - On topic detail/click → "Add Words" button → dialog with search/select from dictionary
  - Select multiple entries → call `batchLinkEntriesToTopic` (max 200)
  - Show success toast: "X words added to '{topic}'"
- **From Dictionary Page** (optional enhancement):
  - When filtering by topic, add "Remove from topic" action on entries
  - Bulk select + "Add to topic" action

**Commit:** `feat(topics): add entry-topic linking from multiple contexts`

---

## Task 4: Inbox GraphQL Layer + Page

**Goal:** Quick capture inbox — list, add, delete notes.

**Context:** Inbox is intentionally simple — a scratchpad for unfamiliar words heard in context. Items have `text` (1-500 chars) and optional `context` (0-2000 chars, e.g., "Heard in podcast about AI"). Items are completely disconnected from dictionary entries. `clearInbox` deletes all items at once.

**What to do:**
- Create `src/graphql/queries/inbox.ts`:
  - `INBOX_ITEMS_QUERY` — list with offset pagination (limit, offset → items + totalCount)
- Create `src/graphql/mutations/inbox.ts`:
  - `CREATE_INBOX_ITEM`, `DELETE_INBOX_ITEM`, `CLEAR_INBOX`
- Define types in `src/types/inbox.ts`:
  - `InboxItem`: `{ id, text, context, createdAt }`
- Replace stub `src/pages/InboxPage.tsx`:
  - **Add form**: text input (required) + context textarea (optional), always visible at top. "Add" button.
  - **Items list**: chronological list of inbox items showing text, context (if any), relative date ("2 hours ago")
  - **Delete**: delete button per item (with or without confirmation — items are lightweight)
  - **Clear all**: "Clear Inbox" button (danger style) with confirmation dialog: "Delete all X items?"
  - **Pagination**: "Load more" or simple offset pagination (max 200 per page per `BUSINESS_RULES.md`)
  - **Workflow hint**: each item could have a "Look up" action that opens CatalogSearch (from Phase 3) pre-filled with the item's text — bridges the gap between inbox and dictionary. This is a UX convenience, not a backend feature.
- Create `src/hooks/useInbox.ts` — wraps query with pagination state

**Commit:** `feat(inbox): implement inbox page with add, delete, clear, and catalog lookup`

---

## Task 5: Loading & Empty States

**Goal:** Polish both features.

**What to do:**
- **Topics loading**: skeleton cards matching topic card layout
- **Topics empty**: "No topics yet. Create one to organize your vocabulary!" with "New Topic" CTA
- **Inbox loading**: skeleton list items
- **Inbox empty**: "Your inbox is empty. Jot down words you hear and look them up later!" — friendly, encouraging
- **Error states**: standard error + retry for both pages

**Commit:** `feat(organization): add loading and empty states for topics and inbox`

---

## Summary

| Task | Description | Key files |
|------|-------------|-----------|
| 1 | Topics GraphQL layer | `src/graphql/queries/topics.ts`, `mutations/topics.ts`, `src/types/topics.ts` |
| 2 | Topics page | `src/pages/TopicsPage.tsx`, `src/components/topics/TopicCard.tsx` |
| 3 | Entry-topic linking | Modify DictionaryEntryPage, add batch linking |
| 4 | Inbox page | `src/graphql/**/inbox.ts`, `src/pages/InboxPage.tsx` |
| 5 | Loading & empty states | Skeletons and empty messages for both features |

**Total:** 5 tasks, ~5 commits
