# Phase 4: Study — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Core of the application — spaced repetition study session. User starts a session, reviews flashcards one by one, grades recall (Again/Hard/Good/Easy), can undo, sees progress, and finishes with a results summary. This is the most interaction-heavy phase.

**Architecture:** Session lifecycle is mutation-driven (start → review N cards → finish). Study queue is a query that returns cards ordered by priority (due first, then new). Frontend manages a local queue, sends review grades one at a time, and fetches more cards when the queue is low. Undo restores the last review within 10 minutes.

**Key references:**
- API: `backend_v4/docs/API.md` — Study section (lines 131–163)
- Study workflow: `docs/business/WORKFLOWS.md` — "Studying Vocabulary" (start session, review, undo, finish)
- Business rules: `docs/business/BUSINESS_RULES.md` — SRS rules (one active session, undo window 10 min, queue size 50 default/200 max, review duration cap 600s), FSRS-5 parameters
- Card state transitions: `docs/business/WORKFLOWS.md` — NEW→LEARNING→REVIEW, REVIEW→RELEARNING→REVIEW
- Design: `frontend-real/design-docs/palette-v3.html` — SRS Flashcard (complete component with word, phonetic, definition, translation, example, meta, SRS buttons), SRS Buttons disabled state

---

## Task 1: Study GraphQL Layer

**Goal:** All queries and mutations for the study domain, typed and ready.

**What to do:**
- Create `src/graphql/queries/study.ts`:
  - `STUDY_QUEUE_QUERY` — fetch cards to review (`studyQueue(limit)`)
  - `CARD_HISTORY_QUERY` — review history for a card
  - `CARD_STATS_QUERY` — card statistics
- Create `src/graphql/mutations/study.ts`:
  - `START_STUDY_SESSION` — start or resume active session
  - `REVIEW_CARD` — submit a grade (AGAIN/HARD/GOOD/EASY + durationMs)
  - `UNDO_REVIEW` — undo last review
  - `FINISH_STUDY_SESSION` — end session, get results
- Define types in `src/types/study.ts`:
  - `StudyCard` — entry data shaped for study (id, text, senses with definitions/translations/examples, card state/due)
  - `ReviewGrade` — enum AGAIN | HARD | GOOD | EASY
  - `ReviewResult` — card state after review (state, stability, difficulty, due, reps, lapses)
  - `SessionResult` — finish result (totalReviews, accuracyRate, gradeCounts)
  - `CardHistory`, `CardStats`

**Commit:** `feat(study): add GraphQL queries, mutations, and types`

---

## Task 2: Study Session Hook

**Goal:** Central hook managing the entire study session lifecycle — queue, current card, timing, undo state.

**Context:** The study flow is stateful: start session → load queue → show card → user grades → send review → show next card → repeat until done. Need to track: current card, cards remaining, time spent on current card, whether undo is available, session status.

**What to do:**
- Create `src/hooks/useStudySession.ts`:
  - **Session lifecycle**: `startSession()`, `finishSession()`, `abandonSession()` (which is just navigating away — session stays active for later)
  - **Queue management**:
    - Fetch initial batch (`studyQueue(limit: 50)`)
    - Track current card index in local state
    - When queue runs low (e.g., < 10 cards left), prefetch next batch
    - Handle empty queue (no due or new cards)
  - **Review flow**:
    - `revealAnswer()` — flip card to show definition/translation/examples
    - `submitReview(grade: ReviewGrade)` — call `reviewCard` mutation with grade + durationMs
    - Track `durationMs` per card (start timer when card shown, stop on grade submit, cap at 600,000ms per `BUSINESS_RULES.md`)
    - After review → advance to next card in queue
  - **Undo**:
    - `undoLastReview()` — call `undoReview`, revert to previous card
    - Track whether undo is available (last reviewed card ID + timestamp, available for 10 min)
    - After undo → show the same card again for re-grading
  - **Progress tracking**:
    - `reviewedCount`, `totalCount`, `newReviewedCount`, `dueReviewedCount`
    - Grade distribution accumulator (again/hard/good/easy counts)
  - **State machine**: `idle` → `loading` → `reviewing` (front) → `reviewing` (back/revealed) → `submitting` → `reviewing` (next card) → `finished`
  - Return: `{ currentCard, isRevealed, progress, canUndo, sessionActive, actions: { reveal, submit, undo, finish } }`

**Commit:** `feat(study): implement study session hook with queue and timing`

---

## Task 3: Flashcard Component

**Goal:** The flashcard UI — the most visually important component in the app.

**Context:** Complete reference in `palette-v3.html` section "SRS Flashcard". The card has two states: front (word + phonetics only) and back (word + phonetics + definition + translation + example + meta). Use the exact Herbarium typography: Lisu Bosa for the word, EB Garamond for phonetics, Space Grotesk for definition, appropriate font for examples based on source type.

**What to do:**
- Create `src/components/study/Flashcard.tsx`:
  - **Front state** (before reveal):
    - Word text in Lisu Bosa font, large (`--type-xl` or bigger)
    - First letter accented in poppy (as shown in `palette-v3.html`: `<span style="color: var(--accent);">e</span>phemeral`)
    - Phonetic transcription (EB Garamond, italic, `--text-tertiary`)
    - Part of speech label
    - Tap/click anywhere or "Show Answer" button to reveal
  - **Back state** (after reveal):
    - Everything from front, plus:
    - Definition (Space Grotesk, `--text-secondary`)
    - Translation (italic, `--text-tertiary`, left border — per `.translation` in palette)
    - Example sentence in styled block (per `.flashcard-example` in palette — sage-light bg, fern left border, EB Garamond for book examples). Choose font by source type if data available.
    - Meta bar: review count, streak/laps count, progress bar (thyme fill)
  - **Card container**: white bg, `--radius-lg`, `--border`, `--elevation-2`, max-width `--container-narrow` (480px), centered
  - **Transition**: smooth reveal animation (e.g., fade-in for the answer section, using `--duration-normal` and `--ease-standard`)
- Props: `card: StudyCard`, `isRevealed: boolean`, `onReveal: () => void`

**Commit:** `feat(study): implement Flashcard component`

---

## Task 4: SRS Grade Buttons

**Goal:** The four grade buttons that appear after answer is revealed.

**Context:** Already have `SrsButtons` base component from Phase 0 Task 4. Now wire it into the study flow with real interval labels from the review response.

**What to do:**
- Enhance `src/components/ui/SrsButtons.tsx` or create `src/components/study/GradeButtons.tsx`:
  - Four buttons: Again (poppy), Hard (goldenrod), Good (cornflower), Easy (thyme)
  - Each shows the predicted next interval as subtitle (e.g., "< 1m", "6m", "10m", "4d")
  - Interval labels: the backend returns the next due date in `reviewCard` response — but the intervals are shown BEFORE the user clicks. Two options:
    - Option A: Show static labels based on card state (server doesn't expose preview intervals)
    - Option B: Don't show intervals, just grade labels
    - Decide based on what data is available. If backend doesn't expose preview intervals, use Option B or show generic indicators.
  - Disabled state while submitting review (prevent double-tap)
  - Keyboard shortcuts: 1=Again, 2=Hard, 3=Good, 4=Easy (for desktop power users)
  - Visual: per `palette-v3.html` `.srs-btn` styles — flex, rounded, white text, hover lift, active scale
- Props: `onGrade: (grade: ReviewGrade) => void`, `disabled: boolean`, `intervals?: { again, hard, good, easy }` (optional)

**Commit:** `feat(study): wire SRS grade buttons with keyboard shortcuts`

---

## Task 5: Study Page Assembly

**Goal:** Complete study page — flashcard + controls + progress.

**What to do:**
- Replace stub `src/pages/StudyPage.tsx`:
  - **Session start**: on mount, call `startSession()` from useStudySession hook
    - If no cards available → show "No cards to review" empty state with link back to dashboard
    - If session starts → begin showing cards
  - **Layout**:
    - Top: progress indicator (e.g., "Card 5 of 32", progress bar)
    - Center: Flashcard component (takes most of the screen)
    - Bottom (revealed): GradeButtons
    - Undo button: small, appears after a review for 10 min (e.g., floating or in top-right, "Undo last" with a subtle timer)
  - **Flow**:
    - Card appears (front) → user taps to reveal → answer appears (back) → grade buttons appear → user grades → next card → repeat
    - After last card in queue → try to fetch more → if no more → auto-finish or prompt to finish
  - **Finish button**: always accessible (e.g., in header), "End Session" — with confirmation if cards remain
  - **Background**: use `--bg-srs` (linen) for the study page background, differentiating it from other pages

**Commit:** `feat(study): implement study page`

---

## Task 6: Session Results Screen

**Goal:** Summary screen after finishing a study session.

**Context:** `finishStudySession` returns `session.result` with: `totalReviews`, `accuracyRate`, `gradeCounts { again, hard, good, easy }`. Accuracy = (good + easy) / total * 100 per `BUSINESS_RULES.md`.

**What to do:**
- Create `src/components/study/SessionResults.tsx`:
  - **Summary stats**:
    - Total cards reviewed
    - Accuracy rate (percentage, with color: thyme if ≥ 80%, goldenrod if 60-79%, poppy if < 60%)
    - Time spent (calculated from accumulated durationMs on client)
  - **Grade distribution**: visual bar or mini chart showing again/hard/good/easy breakdown with semantic colors
  - **Encouragement message** based on performance (optional, simple: "Great session!" / "Keep practicing!")
  - **Actions**:
    - "Study More" button → start new session (if more cards available)
    - "Back to Dashboard" button → navigate to `/dashboard`
- Show this screen inline on StudyPage after `finishSession()` completes (replace the flashcard area)
- Dashboard should refetch on return (already handled by `useDashboard` if it uses `refetchOnWindowFocus` or similar)

**Commit:** `feat(study): add session results screen`

---

## Task 7: Loading, Empty & Edge Cases

**Goal:** Handle all study edge cases gracefully.

**What to do:**
- **Loading**: skeleton flashcard while queue is loading (card-shaped skeleton with pulse)
- **Empty queue**: "All caught up! No cards to review right now." with dashboard link. Check both due and new — if newToday >= limit AND no due cards → this state.
- **Mid-session queue empty**: if all fetched cards reviewed but `fetchMore` returns empty → auto-trigger finish flow
- **Undo edge cases**:
  - Undo button disappears after 10 minutes (use `setTimeout` or check timestamp on render)
  - After undo, the re-shown card should clear the undo state (can't undo an undo)
- **Network error during review**: show error toast, keep card visible, let user retry grade
- **Session already active**: `startSession` is idempotent (returns existing session per `BUSINESS_RULES.md`) — handle transparently
- **Abandon vs finish**: navigating away without finishing leaves session ACTIVE — on next visit to `/study`, hook resumes the session. Consider showing a "You have an unfinished session" prompt on Dashboard (already covered by `activeSession` in Phase 2).

**Commit:** `feat(study): handle loading, empty queue, and edge cases`

---

## Summary

| Task | Description | Key files |
|------|-------------|-----------|
| 1 | GraphQL layer | `src/graphql/queries/study.ts`, `mutations/study.ts`, `src/types/study.ts` |
| 2 | Session hook | `src/hooks/useStudySession.ts` |
| 3 | Flashcard component | `src/components/study/Flashcard.tsx` |
| 4 | Grade buttons | `src/components/study/GradeButtons.tsx` |
| 5 | Study page | `src/pages/StudyPage.tsx` |
| 6 | Results screen | `src/components/study/SessionResults.tsx` |
| 7 | Edge cases | Loading, empty queue, undo timing, network errors |

**Total:** 7 tasks, ~7 commits
