# MyEnglish Frontend

## 1. Project Context

### Product

**MyEnglish** — spaced-repetition vocabulary learning platform. Users build personal English dictionaries, create flashcards, and study them on an algorithm-optimized schedule.

### Target Audience

Non-native English speakers (Russian-speaking primarily), levels CEFR A2–C1.

### Documentation Sources

| Document | Path | Purpose |
| **Domain Model** | `docs/business/DOMAIN_MODEL.md` | Entities (Entry, Sense, Card, Topic, Inbox etc.) and relationships |
| **Business Rules** | `docs/business/BUSINESS_RULES.md` | All limits, validations, formulas (SRS, streak, accuracy) |
| **Workflows** | `docs/business/WORKFLOWS.md` | Step-by-step scenarios: registration, adding words, studying, dashboard |
| **Roles & Access** | `docs/business/ROLES_AND_ACCESS.md` | Permission matrix user/admin, data isolation |
| **Integrations** | `docs/business/INTEGRATIONS.md` | OAuth (Google, Apple), AI enrichment, reference data sources |
| **Open Questions** | `docs/business/UNKNOWNS.md` | Unresolved business questions and assumptions |
| **Backend Architecture** | `backend_v4/docs/ARCHITECTURE.md` | Layered Go service architecture |
| **Backend Components** | `backend_v4/docs/COMPONENTS.md` | Backend module descriptions |
| **Data Flows** | `backend_v4/docs/DATA_FLOW.md` | How data flows through the backend |
| **Architecture Decisions** | `backend_v4/docs/DECISIONS.md` | ADRs — reasoning behind key decisions |
| **Quickstart** | `backend_v4/docs/QUICKSTART.md` | How to run the backend locally |

### Visual System

**Herbarium Design System**
 botanical "pressed herbarium" theme. Muted nature-inspired palette. White backgrounds. Accent — poppy color (muted vermillion).


**6 fonts** :

// TODO: заменить space grotesk на что-то более подходящее для primary UI
- Space Grotesk — primary UI text
- Orelega One — headings, words on flashcards
- EB Garamond — book example sentences
- Courier Prime — movie/TV subtitle examples
// TODO: подобрать другой шрифт для примеров из песен
- Caveat — song lyric examples

### Tech Stack
- React + TypeScript + Vite
- Tailwind CSS
// TODO: найти какие еще готовые библиотеки можно использовать
- shadcn/ui
- Apollo Client (GraphQL)
- React Router (routing)

### API

- REST: auth (`/auth/*`), health (`/live`, `/ready`, `/health`), admin (`/admin/*`)
- GraphQL: all app logic (`POST /query`) — dictionary, study, topics, inbox, profile
- JWT: access token (15 min) + refresh token (30 days, rotation)
- Pagination: cursor-based (dictionary, cardHistory) + offset-based (admin, inbox)

### Key User Flows

(Detailed in `docs/business/WORKFLOWS.md`)

1. **Registration/Login** — email+password or Google OAuth
2. **Dashboard** — stats: due/new/streak/reviewed today/status counts
3. **Dictionary** — catalog search, add words, CRUD entries/senses/translations/examples
4. **Study** — start session → card queue → grade (Again/Hard/Good/Easy) → undo → finish → results
5. **Organization** — topics (word grouping) + inbox (quick notes)
6. **Settings** — profile, new cards/day, desired retention, timezone
7. **Admin** — user management, enrichment queue

---

## 2. Implementation Phases

### Phase 0: Foundation

**Goal**: Runnable empty project with all infrastructure. After this phase, any feature can be worked on.

**Deliverable**: Navigate to any route, see navigation, all base components work.

---

### Phase 1: Auth

**Goal**: User can register, log in, and log out.

**Scope**:

- Login page (email + password)
- Register page (email, username, password)
- Google OAuth flow ("Sign in with Google" button)
- JWT handling: store access/refresh tokens, automatic refresh on expiry
- Logout (revoke all sessions)
- Redirect unauthorized users to login
- Form validation per `BUSINESS_RULES.md` (email format, password 8-72 chars, username 2-50 chars)
- Error handling: rate limiting (429), validation errors, generic unauthorized

**API**: REST — `POST /auth/register`, `POST /auth/login`, `POST /auth/login/password`, `POST /auth/refresh`, `POST /auth/logout`

**Deliverable**: Full auth cycle, after login — empty dashboard.

---

### Phase 2: Dashboard

**Goal**: Home page after login — learning progress overview.

**Scope**:

- Stat widgets: Due Count, New Count, Reviewed Today, New Today, Overdue Count, Streak
- Card status distribution (New / Learning / Review / Relearning)
- "Start Study" button (links to Phase 4, stub until then)
- Responsive layout: mobile (stacked cards) + desktop (grid)

**API**: GraphQL — `dashboard` query

**Deliverable**: After login, user sees their progress and can start studying.

---

### Phase 3: Dictionary

**Goal**: User can manage their personal dictionary — the most voluminous phase.

**Scope**:

- **Word list**: cursor pagination, search, sort (created_at, updated_at), filters (hasCard, partOfSpeech, topicId, status)
- очень важно сделать так, чтобы при поиске страница не обновлялась полность и поле ввода не деактивировалось после ввода пользователем символов.
- **Word card in list**: Vocab Card — word, definition, status (pill), part of speech (tag) 
// TODO: перенести обзор карточки слова из прошлой версии фронтенда
- **Word detail page**: full Entry view with Senses, Translations, Examples, User Images, Topics, Card info
- **Add from catalog**: пользователь выбирает слово из поиска -> открывается карточка этого слова. (пока что пусть пользователь просто выбирает какие senses выбрать для себя, но в будущем надо чтобы можно было выбирать любую инфу из ref - что оставить а что убрать)
- **Add custom word**: form with text, senses, definitions, translations, examples
- **Edit**: CRUD for Senses, Translations, Examples (add/update/delete/reorder)
- **Delete**: soft delete with confirmation
- **Trash**: deleted entries list, restore
- **Bulk import**: upload word list, display results (created/skipped/errors)
- **Card creation**: "Create Card" button for entries without card, batch create

**API**: GraphQL — `dictionary`, `dictionaryEntry`, `searchCatalog`, `previewRefEntry`, `createEntryFromCatalog`, `createEntryCustom`, `deleteEntry`, `restoreEntry`, `importEntries`, `deletedEntries`, `addSense`, `updateSense`, `deleteSense`, `reorderSenses`, `addTranslation`, `updateTranslation`, `deleteTranslation`, `addExample`, `updateExample`, `deleteExample`, `addUserImage`, `deleteUserImage`, `createCard`, `batchCreateCards`, `deleteCard`

**Deliverable**: Full-featured dictionary — add, edit, delete, search, filter.

---

### Phase 4: Study

**Goal**: Application core — spaced repetition study session.

**Scope**:

- **Start session**: button from Dashboard or navigation
- **Flashcard UI**: component — word, phonetics, definition, translation, examples
- **"Show answer" mechanic**: first show word, on click — definition + translation + examples
- **SRS buttons**: Again / Hard / Good / Easy with intervals
- **Undo**: undo last review button (within 10 min window)
- **Session progress**: progress bar, card counter
- **Session finish**: results screen — total reviews, accuracy rate, grade distribution, new vs due
- **Abandon session**: option to interrupt session
- **Queue management**: load next batch of cards when current batch runs out

**API**: GraphQL — `studyQueue`, `startStudySession`, `reviewCard`, `undoReview`, `finishStudySession`, `cardHistory`, `cardStats`

**Deliverable**: Full study cycle — from start to results with beautiful flashcard and SRS buttons.

---

### Phase 5: Organization (Topics + Inbox)

**Goal**: Dictionary organization tools.

**Scope**:

- **Topics**: topic list with word counts, CRUD (create/update/delete), link words to topic (individual + batch up to 200), filter dictionary by topic
- **Inbox**: note list with pagination, add (text + context), delete, clear all

**API**: GraphQL — `topics`, `createTopic`, `updateTopic`, `deleteTopic`, `linkEntryToTopic`, `batchLinkEntriesToTopic`, `inboxItems`, `createInboxItem`, `deleteInboxItem`, `clearInbox`

**Deliverable**: User can group words by themes and keep quick notes.

---

### Phase 6: Settings & Profile

**Goal**: Account and learning parameter management.

**Scope**:

- **Profile**: name, avatar (URL), email (read-only), username (read-only)
- **Study settings**: new cards per day (1-999), reviews per day goal (1-9999), max interval (1-36500), desired retention (0.70-0.99) — with explanations of what each parameter means
- **Timezone**: IANA timezone picker
- **Validation** per `BUSINESS_RULES.md` rules

**API**: GraphQL — `me`, `userSettings`, `updateProfile`, `updateSettings`

**Deliverable**: User can customize their learning experience.

---

### Phase 7: Admin Panel

**Goal**: Administrator panel for platform management.

**Scope**:

- **User management**: list (offset pagination), role change user↔admin (with self-demote protection)
- **Enrichment Queue**: stats (pending/processing/done/failed/total), queue list with status filter, retry failed, reset stuck processing, manual enqueue
- Access restricted to role=admin, 403 for regular users
- Separate navigation section or separate layout

**API**: REST — `GET /admin/users`, `PUT /admin/users/{id}/role`, `GET /admin/enrichment/stats`, `GET /admin/enrichment/queue`, `POST /admin/enrichment/enqueue`, `POST /admin/enrichment/retry`, `POST /admin/enrichment/reset-processing`

**Deliverable**: Admin can manage users and AI enrichment.

---

### Phase 8: Polish & Edge Cases

**Goal**: Production-quality finish.

**Scope**:

- **Error handling**: global error boundary, retry logic, offline state
- **Loading states**: Skeleton loaders on all pages
- **Empty states**: when no words, no cards, no topics
- **Animations**: 
- **Toast notifications**: success/error for all mutations
- **Optimistic updates**: for frequently used actions (review card, delete entry)
- **Performance**: code splitting per route, lazy loading

**Deliverable**: Production-ready application.
