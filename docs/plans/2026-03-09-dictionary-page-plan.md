# Dictionary Page Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a full-featured dictionary page with grid/table views, search & filters, quick-preview Sheet, full detail page, edit modal, and delete with undo.

**Architecture:** DictionaryPage orchestrates filter state and delegates rendering to WordCardGrid or WordTable. WordSheet provides quick preview; DictionaryEntryPage provides full view. WordEditDialog handles editing from both contexts. All data flows through Apollo Client GraphQL queries. i18n from day one.

**Tech Stack:** React 19, TypeScript, Apollo Client 4, shadcn/ui, Tailwind CSS (Herbarium tokens), Lucide icons, Framer Motion (page transitions only), react-i18next, Sonner (toasts).

**Design doc:** `docs/plans/2026-03-09-dictionary-page-design.md`
**Design system:** `docs/plans/frontend-design.md`

**Existing patterns to follow:**
- GraphQL queries: see `src/graphql/queries/me.ts` for structure
- i18n: see `src/i18n/index.ts` — import JSON, register namespace
- Components: see `src/components/common/StatusPill.tsx` for Herbarium token usage
- UI primitives: `src/components/ui/` — shadcn/ui components

---

### Task 1: Types & GraphQL

**Files:**
- Create: `src/types/dictionary.ts`
- Create: `src/graphql/queries/dictionary.ts`

**Step 1: Create dictionary types**

Create `src/types/dictionary.ts`:

```typescript
export type UUID = string

export type PartOfSpeech =
  | 'NOUN'
  | 'VERB'
  | 'ADJECTIVE'
  | 'ADVERB'
  | 'PRONOUN'
  | 'PREPOSITION'
  | 'CONJUNCTION'
  | 'INTERJECTION'
  | 'PHRASE'
  | 'IDIOM'
  | 'OTHER'

export type SortBy = 'TEXT' | 'CREATED_AT' | 'UPDATED_AT'
export type SortDir = 'ASC' | 'DESC'

export interface Translation {
  id: UUID
  text: string
}

export interface Example {
  id: UUID
  sentence: string
  translation?: string
}

export interface Sense {
  id: UUID
  definition?: string
  partOfSpeech?: PartOfSpeech
  translations: Translation[]
  examples: Example[]
}

export interface Image {
  id: UUID
  url: string
  caption?: string
}

export interface Pronunciation {
  id: UUID
  audioUrl?: string
  transcription: string
  region?: string
}

export interface Topic {
  id: UUID
  name: string
  description?: string
}

export interface DictionaryEntry {
  id: UUID
  text: string
  textNormalized: string
  notes?: string
  senses: Sense[]
  images: Image[]
  pronunciations: Pronunciation[]
  topics: Topic[]
  createdAt: string
  updatedAt: string
}

export interface WordFilter {
  search?: string
  topicIDs?: UUID[]
  partOfSpeech?: PartOfSpeech
  sortBy?: SortBy
  sortDir?: SortDir
  limit?: number
}
```

**Step 2: Create GraphQL queries and mutations**

Create `src/graphql/queries/dictionary.ts`:

```typescript
import { gql } from '@apollo/client'

export const GET_DICTIONARY = gql`
  query GetDictionary($filter: WordFilter) {
    dictionary(filter: $filter) {
      id
      text
      textNormalized
      senses {
        id
        definition
        partOfSpeech
        translations {
          id
          text
        }
        examples {
          id
          sentence
          translation
        }
      }
      images {
        id
        url
        caption
      }
      pronunciations {
        id
        audioUrl
        transcription
        region
      }
      topics {
        id
        name
      }
      createdAt
      updatedAt
    }
  }
`

export const GET_DICTIONARY_ENTRY = gql`
  query GetDictionaryEntry($id: UUID!) {
    dictionaryEntry(id: $id) {
      id
      text
      textNormalized
      notes
      senses {
        id
        definition
        partOfSpeech
        translations {
          id
          text
        }
        examples {
          id
          sentence
          translation
        }
      }
      images {
        id
        url
        caption
      }
      pronunciations {
        id
        audioUrl
        transcription
        region
      }
      topics {
        id
        name
        description
      }
      createdAt
      updatedAt
    }
  }
`

export const GET_TOPICS = gql`
  query GetTopics {
    topics {
      id
      name
      description
    }
  }
`

export const UPDATE_WORD = gql`
  mutation UpdateWord($id: UUID!, $input: UpdateWordInput!) {
    updateWord(id: $id, input: $input) {
      id
      text
      textNormalized
      notes
      senses {
        id
        definition
        partOfSpeech
        translations {
          id
          text
        }
        examples {
          id
          sentence
          translation
        }
      }
      images {
        id
        url
        caption
      }
      pronunciations {
        id
        audioUrl
        transcription
        region
      }
      topics {
        id
        name
        description
      }
      updatedAt
    }
  }
`

export const DELETE_WORD = gql`
  mutation DeleteWord($id: UUID!) {
    deleteWord(id: $id)
  }
`
```

**Step 3: Verify build**

Run: `cd frontend-real && npx tsc --noEmit`
Expected: No errors related to new files.

**Step 4: Commit**

```bash
git add src/types/dictionary.ts src/graphql/queries/dictionary.ts
git commit -m "feat(dictionary): add types and GraphQL queries"
```

---

### Task 2: i18n — dictionary namespace

**Files:**
- Create: `src/i18n/locales/en/dictionary.json`
- Create: `src/i18n/locales/ru/dictionary.json`
- Modify: `src/i18n/index.ts` — add dictionary namespace imports

**Step 1: Create English translations**

Create `src/i18n/locales/en/dictionary.json`:

```json
{
  "page.title": "Dictionary",
  "search.placeholder": "Search words...",
  "filter.topics": "Topics",
  "filter.partOfSpeech": "Part of speech",
  "filter.clearAll": "Clear filters",
  "sort.alphabetical": "Alphabetical",
  "sort.created": "Date created",
  "sort.updated": "Date updated",
  "sort.asc": "A → Z",
  "sort.desc": "Z → A",
  "view.grid": "Grid view",
  "view.table": "Table view",
  "count.words_one": "{{count}} word",
  "count.words_other": "{{count}} words",
  "empty.title": "Your dictionary is empty",
  "empty.noResults": "No words match your filters",
  "table.word": "Word",
  "table.translations": "Translations",
  "table.definition": "Definition",
  "table.partOfSpeech": "Part of speech",
  "table.topics": "Topics",
  "table.updated": "Updated",
  "sheet.openFull": "Open full page",
  "sheet.edit": "Edit",
  "sheet.delete": "Delete",
  "edit.title": "Edit word",
  "edit.word": "Word",
  "edit.notes": "Notes",
  "edit.topics": "Topics",
  "edit.save": "Save",
  "edit.saving": "Saving...",
  "edit.cancel": "Cancel",
  "edit.senses": "Meanings",
  "edit.addSense": "Add meaning",
  "edit.removeSense": "Remove meaning",
  "edit.definition": "Definition",
  "edit.partOfSpeech": "Part of speech",
  "edit.translations": "Translations",
  "edit.addTranslation": "Add translation",
  "edit.examples": "Examples",
  "edit.addExample": "Add example",
  "edit.sentence": "Sentence",
  "edit.translation": "Translation",
  "edit.media": "Media",
  "edit.images": "Images",
  "edit.addImage": "Add image",
  "edit.imageUrl": "Image URL",
  "edit.caption": "Caption",
  "edit.pronunciations": "Pronunciations",
  "edit.addPronunciation": "Add pronunciation",
  "edit.transcription": "Transcription",
  "edit.audioUrl": "Audio URL",
  "edit.region": "Region",
  "delete.toast": "Word deleted",
  "delete.undo": "Undo",
  "delete.confirm": "Delete word?",
  "error.loadFailed": "Failed to load dictionary. Check your connection.",
  "error.saveFailed": "Failed to save changes. Try again.",
  "error.deleteFailed": "Failed to delete word. Try again.",
  "error.tryAgain": "Try again",
  "entry.backToDict": "Back to Dictionary",
  "pos.NOUN": "Noun",
  "pos.VERB": "Verb",
  "pos.ADJECTIVE": "Adjective",
  "pos.ADVERB": "Adverb",
  "pos.PRONOUN": "Pronoun",
  "pos.PREPOSITION": "Preposition",
  "pos.CONJUNCTION": "Conjunction",
  "pos.INTERJECTION": "Interjection",
  "pos.PHRASE": "Phrase",
  "pos.IDIOM": "Idiom",
  "pos.OTHER": "Other",
  "actions.open": "Open",
  "actions.edit": "Edit",
  "actions.delete": "Delete"
}
```

**Step 2: Create Russian translations**

Create `src/i18n/locales/ru/dictionary.json`:

```json
{
  "page.title": "Словарь",
  "search.placeholder": "Поиск слов...",
  "filter.topics": "Темы",
  "filter.partOfSpeech": "Часть речи",
  "filter.clearAll": "Сбросить фильтры",
  "sort.alphabetical": "По алфавиту",
  "sort.created": "Дата создания",
  "sort.updated": "Дата обновления",
  "sort.asc": "А → Я",
  "sort.desc": "Я → А",
  "view.grid": "Карточки",
  "view.table": "Таблица",
  "count.words_one": "{{count}} слово",
  "count.words_few": "{{count}} слова",
  "count.words_many": "{{count}} слов",
  "empty.title": "Ваш словарь пуст",
  "empty.noResults": "Нет слов по вашим фильтрам",
  "table.word": "Слово",
  "table.translations": "Переводы",
  "table.definition": "Определение",
  "table.partOfSpeech": "Часть речи",
  "table.topics": "Темы",
  "table.updated": "Обновлено",
  "sheet.openFull": "Открыть полностью",
  "sheet.edit": "Редактировать",
  "sheet.delete": "Удалить",
  "edit.title": "Редактирование слова",
  "edit.word": "Слово",
  "edit.notes": "Заметки",
  "edit.topics": "Темы",
  "edit.save": "Сохранить",
  "edit.saving": "Сохранение...",
  "edit.cancel": "Отмена",
  "edit.senses": "Значения",
  "edit.addSense": "Добавить значение",
  "edit.removeSense": "Удалить значение",
  "edit.definition": "Определение",
  "edit.partOfSpeech": "Часть речи",
  "edit.translations": "Переводы",
  "edit.addTranslation": "Добавить перевод",
  "edit.examples": "Примеры",
  "edit.addExample": "Добавить пример",
  "edit.sentence": "Предложение",
  "edit.translation": "Перевод",
  "edit.media": "Медиа",
  "edit.images": "Изображения",
  "edit.addImage": "Добавить изображение",
  "edit.imageUrl": "URL изображения",
  "edit.caption": "Подпись",
  "edit.pronunciations": "Произношение",
  "edit.addPronunciation": "Добавить произношение",
  "edit.transcription": "Транскрипция",
  "edit.audioUrl": "URL аудио",
  "edit.region": "Регион",
  "delete.toast": "Слово удалено",
  "delete.undo": "Отменить",
  "delete.confirm": "Удалить слово?",
  "error.loadFailed": "Не удалось загрузить словарь. Проверьте соединение.",
  "error.saveFailed": "Не удалось сохранить изменения. Попробуйте снова.",
  "error.deleteFailed": "Не удалось удалить слово. Попробуйте снова.",
  "error.tryAgain": "Попробовать снова",
  "entry.backToDict": "Назад к словарю",
  "pos.NOUN": "Существительное",
  "pos.VERB": "Глагол",
  "pos.ADJECTIVE": "Прилагательное",
  "pos.ADVERB": "Наречие",
  "pos.PRONOUN": "Местоимение",
  "pos.PREPOSITION": "Предлог",
  "pos.CONJUNCTION": "Союз",
  "pos.INTERJECTION": "Междометие",
  "pos.PHRASE": "Фраза",
  "pos.IDIOM": "Идиома",
  "pos.OTHER": "Другое",
  "actions.open": "Открыть",
  "actions.edit": "Редактировать",
  "actions.delete": "Удалить"
}
```

**Step 3: Register dictionary namespace in i18n config**

Modify `src/i18n/index.ts`:
- Add imports: `import enDictionary from './locales/en/dictionary.json'` and `import ruDictionary from './locales/ru/dictionary.json'`
- Add to resources: `en: { ..., dictionary: enDictionary }`, `ru: { ..., dictionary: ruDictionary }`

```typescript
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import LanguageDetector from 'i18next-browser-languagedetector'

import enAuth from './locales/en/auth.json'
import enValidation from './locales/en/validation.json'
import enDictionary from './locales/en/dictionary.json'
import ruAuth from './locales/ru/auth.json'
import ruValidation from './locales/ru/validation.json'
import ruDictionary from './locales/ru/dictionary.json'

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      en: { auth: enAuth, validation: enValidation, dictionary: enDictionary },
      ru: { auth: ruAuth, validation: ruValidation, dictionary: ruDictionary },
    },
    fallbackLng: 'en',
    defaultNS: 'auth',
    interpolation: { escapeValue: false },
    detection: {
      order: ['localStorage', 'navigator'],
      lookupLocalStorage: 'i18nextLng',
      caches: ['localStorage'],
    },
  })

export default i18n
```

**Step 4: Verify build**

Run: `cd frontend-real && npx tsc --noEmit`

**Step 5: Commit**

```bash
git add src/i18n/
git commit -m "feat(dictionary): add i18n dictionary namespace (en + ru)"
```

---

### Task 3: Install shadcn Sheet component

**Files:**
- Create: `src/components/ui/sheet.tsx`

Sheet is not yet installed. It's built on `@radix-ui/react-dialog` which is already in dependencies.

**Step 1: Install Sheet via shadcn CLI**

Run: `cd frontend-real && npx shadcn@latest add sheet`

If CLI doesn't work (no components.json), manually create `src/components/ui/sheet.tsx` with the standard shadcn Sheet implementation based on @radix-ui/react-dialog. The Sheet component provides: Sheet, SheetTrigger, SheetContent (with side variants: top/bottom/left/right), SheetHeader, SheetFooter, SheetTitle, SheetDescription, SheetClose.

**Step 2: Verify build**

Run: `cd frontend-real && npx tsc --noEmit`

**Step 3: Commit**

```bash
git add src/components/ui/sheet.tsx
git commit -m "feat(ui): add shadcn Sheet component"
```

---

### Task 4: Custom hooks — useDictionary, useWordDetail, useDeleteWord

**Files:**
- Create: `src/hooks/useDictionary.ts`
- Create: `src/hooks/useWordDetail.ts`
- Create: `src/hooks/useDeleteWord.ts`

**Step 1: Create useDictionary hook**

Create `src/hooks/useDictionary.ts`:

```typescript
import { useQuery } from '@apollo/client'
import { GET_DICTIONARY, GET_TOPICS } from '@/graphql/queries/dictionary'
import type { DictionaryEntry, Topic, WordFilter } from '@/types/dictionary'

interface DictionaryData {
  dictionary: DictionaryEntry[]
}

interface TopicsData {
  topics: Topic[]
}

export function useDictionary(filter: WordFilter) {
  const { data, loading, error, refetch } = useQuery<DictionaryData>(GET_DICTIONARY, {
    variables: { filter },
  })

  const { data: topicsData } = useQuery<TopicsData>(GET_TOPICS)

  return {
    entries: data?.dictionary ?? [],
    topics: topicsData?.topics ?? [],
    loading,
    error,
    refetch,
  }
}
```

**Step 2: Create useWordDetail hook**

Create `src/hooks/useWordDetail.ts`:

```typescript
import { useQuery } from '@apollo/client'
import { GET_DICTIONARY_ENTRY } from '@/graphql/queries/dictionary'
import type { DictionaryEntry } from '@/types/dictionary'

interface DictionaryEntryData {
  dictionaryEntry: DictionaryEntry
}

export function useWordDetail(id: string | undefined) {
  const { data, loading, error } = useQuery<DictionaryEntryData>(GET_DICTIONARY_ENTRY, {
    variables: { id },
    skip: !id,
  })

  return {
    entry: data?.dictionaryEntry,
    loading,
    error,
  }
}
```

**Step 3: Create useDeleteWord hook**

Create `src/hooks/useDeleteWord.ts`:

This hook handles optimistic deletion + undo via Sonner toast.

```typescript
import { useMutation } from '@apollo/client'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import { DELETE_WORD, GET_DICTIONARY } from '@/graphql/queries/dictionary'
import type { DictionaryEntry } from '@/types/dictionary'

export function useDeleteWord() {
  const { t } = useTranslation('dictionary')
  const [deleteWord] = useMutation(DELETE_WORD)

  const handleDelete = (entry: DictionaryEntry) => {
    let undone = false

    deleteWord({
      variables: { id: entry.id },
      optimisticResponse: { deleteWord: true },
      update(cache) {
        cache.modify({
          fields: {
            dictionary(existingRefs: DictionaryEntry[] = [], { readField }) {
              return existingRefs.filter(
                (ref) => readField('id', ref) !== entry.id
              )
            },
          },
        })
      },
    }).catch(() => {
      if (!undone) {
        toast.error(t('error.deleteFailed'))
      }
    })

    toast(t('delete.toast'), {
      action: {
        label: t('delete.undo'),
        onClick: () => {
          undone = true
          // Refetch to restore the word since we can't truly undo server-side
          // The backend should support soft-delete or the undo window should
          // cancel the mutation before it completes
        },
      },
      duration: 5000,
    })
  }

  return { deleteWord: handleDelete }
}
```

Note: The undo implementation depends on backend support. If the backend doesn't support soft-delete, we refetch the dictionary list. The implementer should check the actual backend behavior and adjust accordingly.

**Step 4: Verify build**

Run: `cd frontend-real && npx tsc --noEmit`

**Step 5: Commit**

```bash
git add src/hooks/useDictionary.ts src/hooks/useWordDetail.ts src/hooks/useDeleteWord.ts
git commit -m "feat(dictionary): add data hooks (useDictionary, useWordDetail, useDeleteWord)"
```

---

### Task 5: SearchToolbar component

**Files:**
- Create: `src/components/dictionary/SearchToolbar.tsx`

**Dependencies:** Task 1 (types), Task 2 (i18n)

This is the horizontal toolbar with search, filters, sort, view toggle, and result count.

**Step 1: Create SearchToolbar**

Create `src/components/dictionary/SearchToolbar.tsx`.

Props interface:

```typescript
import type { PartOfSpeech, SortBy, SortDir, Topic, UUID } from '@/types/dictionary'

type ViewMode = 'grid' | 'table'

interface SearchToolbarProps {
  search: string
  onSearchChange: (value: string) => void
  selectedTopicIds: UUID[]
  onTopicIdsChange: (ids: UUID[]) => void
  selectedPartOfSpeech: PartOfSpeech | null
  onPartOfSpeechChange: (pos: PartOfSpeech | null) => void
  sortBy: SortBy
  sortDir: SortDir
  onSortChange: (sortBy: SortBy, sortDir: SortDir) => void
  viewMode: ViewMode
  onViewModeChange: (mode: ViewMode) => void
  topics: Topic[]
  resultCount: number
}
```

Implementation details:
- Search: `<Input>` with `<Search>` icon (lucide), controlled via props. Debounce happens in the parent (DictionaryPage).
- Topics filter: `<DropdownMenu>` with `<DropdownMenuCheckboxItem>` for each topic. Trigger button shows "Topics" + badge with count if any selected.
- Part of speech filter: `<DropdownMenu>` with `<DropdownMenuCheckboxItem>` for each POS value. Only one selectable at a time (radio-like), but use null for "all".
- Sort: `<DropdownMenu>` with radio items for sortBy + separator + ASC/DESC toggle.
- View toggle: two `<Button variant="ghost" size="icon">` with `<LayoutGrid>` and `<List>` icons. Active one gets `text-poppy bg-poppy-light`.
- Result count: `<span className="text-sm text-text-secondary">` using `t('count.words', { count })`.

Layout: `flex items-center gap-3 flex-wrap`. Search input takes `flex-1 min-w-48`. Filters/sort/toggle group on the right.

**Step 2: Verify build**

Run: `cd frontend-real && npx tsc --noEmit`

**Step 3: Verify visually**

Temporarily render `<SearchToolbar>` in DictionaryPage with hardcoded props to check layout, then remove.

Run: `cd frontend-real && npm run dev` — open http://localhost:5173/dictionary

**Step 4: Commit**

```bash
git add src/components/dictionary/SearchToolbar.tsx
git commit -m "feat(dictionary): add SearchToolbar component"
```

---

### Task 6: WordCardGrid component

**Files:**
- Create: `src/components/dictionary/WordCardGrid.tsx`

**Dependencies:** Task 1 (types), Task 2 (i18n)

Grid view for dictionary entries. Responsive grid: `grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4`.

**Step 1: Create WordCardGrid**

Create `src/components/dictionary/WordCardGrid.tsx`.

Props:

```typescript
interface WordCardGridProps {
  entries: DictionaryEntry[]
  loading: boolean
  onWordClick: (id: string) => void
}
```

Each card:
- Uses `<Card>` from shadcn with `border border-border-default hover:border-poppy transition-colors duration-150 cursor-pointer`.
- Word text: `<h3 className="font-orelega text-xl text-text-primary">`.
- Transcription (first pronunciation if exists): `<span className="font-serif text-sm text-text-secondary">` showing `/transcription/`.
- First sense preview: POS as `<Badge variant="secondary">` with `t('pos.NOUN')` etc. + comma-joined translations.
- Topic badges: max 3 shown as `<Badge variant="outline" className="text-xs">`, "+N" if more.
- Click handler on the card calls `onWordClick(entry.id)`.

Loading state: 6 skeleton cards matching the card layout (Skeleton component with pulse).

**Step 2: Verify build**

Run: `cd frontend-real && npx tsc --noEmit`

**Step 3: Commit**

```bash
git add src/components/dictionary/WordCardGrid.tsx
git commit -m "feat(dictionary): add WordCardGrid component"
```

---

### Task 7: WordTable component

**Files:**
- Create: `src/components/dictionary/WordTable.tsx`

**Dependencies:** Task 1 (types), Task 2 (i18n)

Table view with sticky first column and row actions.

**Step 1: Create WordTable**

Create `src/components/dictionary/WordTable.tsx`.

Props:

```typescript
interface WordTableProps {
  entries: DictionaryEntry[]
  loading: boolean
  sortBy: SortBy
  sortDir: SortDir
  onSortChange: (sortBy: SortBy, sortDir: SortDir) => void
  onWordClick: (id: string) => void
  onEditClick: (id: string) => void
  onDeleteClick: (entry: DictionaryEntry) => void
}
```

Implementation:
- Wrapper: `<div className="overflow-x-auto border border-border-default rounded-md">`.
- `<table className="w-full text-sm">`.
- Sticky first column: `sticky left-0 bg-bg-card z-[1]` on the Word `<th>` and `<td>`.
- Sortable headers (Word → TEXT, Updated → UPDATED_AT): click toggles direction, show `<ArrowUp>` or `<ArrowDown>` icon.
- Row click → `onWordClick(id)`.
- Last column: `<DropdownMenu>` trigger with `<MoreHorizontal>` icon. Items: Open (`onWordClick`), Edit (`onEditClick`), Delete (`onDeleteClick`) with `text-poppy` color for delete.
- Translations column: comma-joined from first sense, truncated with `truncate max-w-48`.
- Definition column: from first sense, truncated with `truncate max-w-64`.
- Part of speech: `<Badge>` from first sense.
- Topics: flex row of badges, max 2 + "+N".
- Updated: formatted date (`new Date(updatedAt).toLocaleDateString()`).

Loading state: 8 skeleton rows.

**Step 2: Verify build**

Run: `cd frontend-real && npx tsc --noEmit`

**Step 3: Commit**

```bash
git add src/components/dictionary/WordTable.tsx
git commit -m "feat(dictionary): add WordTable component"
```

---

### Task 8: WordOverview component

**Files:**
- Create: `src/components/dictionary/WordOverview.tsx`

**Dependencies:** Task 1 (types), Task 2 (i18n)

Reusable read-only display of a word's full information. Used in both WordSheet and DictionaryEntryPage.

**Step 1: Create WordOverview**

Create `src/components/dictionary/WordOverview.tsx`.

Props:

```typescript
interface WordOverviewProps {
  entry: DictionaryEntry
  className?: string
}
```

Sections (top to bottom):
1. **Header**: Word in `font-orelega text-4xl`. Below it: pronunciation block — transcription in `font-serif text-text-secondary` with region badge if present. Audio play button (`<Volume2>` icon) if audioUrl exists — uses `new Audio(url).play()` on click.
2. **Topics**: Row of `<Badge variant="outline">` for each topic.
3. **Senses**: Numbered list. Each sense:
   - POS badge (`<Badge variant="secondary">`)
   - Definition in text-text-primary
   - Translations as comma-separated text-text-secondary
   - Examples: each in `<blockquote>` with font-serif italic, translation below in text-text-secondary text-sm
4. **Images**: If any, grid of images `grid-cols-2 gap-2`, each `<img>` with `rounded-md object-cover aspect-square`, caption below in text-xs text-text-secondary. Use `loading="lazy"`. On error, hide the image.
5. **Notes**: If notes exist, show with a `<Separator>` above, label "Notes" in text-sm font-medium, content in text-text-secondary.

**Step 2: Verify build**

Run: `cd frontend-real && npx tsc --noEmit`

**Step 3: Commit**

```bash
git add src/components/dictionary/WordOverview.tsx
git commit -m "feat(dictionary): add WordOverview component"
```

---

### Task 9: WordSheet component

**Files:**
- Create: `src/components/dictionary/WordSheet.tsx`

**Dependencies:** Task 3 (Sheet UI), Task 4 (useWordDetail), Task 8 (WordOverview)

**Step 1: Create WordSheet**

Create `src/components/dictionary/WordSheet.tsx`.

Props:

```typescript
interface WordSheetProps {
  wordId: string | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onOpenFullPage: (id: string) => void
  onEdit: (id: string) => void
  onDelete: (entry: DictionaryEntry) => void
}
```

Implementation:
- `<Sheet open={open} onOpenChange={onOpenChange}>` with `<SheetContent side="right" className="w-full sm:max-w-lg overflow-y-auto">`.
- Uses `useWordDetail(wordId)` to fetch data.
- Loading: skeleton blocks inside SheetContent.
- Error: inline error message with retry.
- Content: `<SheetHeader>` with `<SheetTitle>` (word text), then `<WordOverview entry={entry}>`.
- Footer: three buttons:
  - "Open full page" — `<Button variant="outline">` with `<ExternalLink>` icon, calls `onOpenFullPage(id)` (navigates to `/dictionary/:id`).
  - "Edit" — `<Button variant="outline">` with `<Pencil>` icon, calls `onEdit(id)`.
  - "Delete" — `<Button variant="ghost" className="text-poppy hover:text-poppy-hover">` with `<Trash2>` icon, calls `onDelete(entry)`.

**Step 2: Verify build**

Run: `cd frontend-real && npx tsc --noEmit`

**Step 3: Commit**

```bash
git add src/components/dictionary/WordSheet.tsx
git commit -m "feat(dictionary): add WordSheet component"
```

---

### Task 10: DictionaryPage — main page wiring

**Files:**
- Modify: `src/pages/DictionaryPage.tsx` (overwrite stub)

**Dependencies:** Tasks 4-9

This is the orchestrator. Manages all state and wires components together.

**Step 1: Implement DictionaryPage**

Overwrite `src/pages/DictionaryPage.tsx`.

State:
```typescript
const [search, setSearch] = useState('')
const [debouncedSearch, setDebouncedSearch] = useState('')
const [selectedTopicIds, setSelectedTopicIds] = useState<string[]>([])
const [selectedPOS, setSelectedPOS] = useState<PartOfSpeech | null>(null)
const [sortBy, setSortBy] = useState<SortBy>('TEXT')
const [sortDir, setSortDir] = useState<SortDir>('ASC')
const [viewMode, setViewMode] = useState<'grid' | 'table'>(() =>
  (localStorage.getItem('dictionary-view') as 'grid' | 'table') || 'grid'
)
const [selectedWordId, setSelectedWordId] = useState<string | null>(null)
const [editWordId, setEditWordId] = useState<string | null>(null)
```

Debounce search with `useEffect` + `setTimeout` (500ms).

Persist viewMode to localStorage on change.

Build `WordFilter` from state, pass to `useDictionary(filter)`.

Layout:
```tsx
<div className="space-y-6">
  <h1 className="text-3xl font-bold text-text-primary">{t('page.title')}</h1>
  <SearchToolbar ... />
  {error && <ErrorBlock />}
  {!error && viewMode === 'grid' && <WordCardGrid ... />}
  {!error && viewMode === 'table' && <WordTable ... />}
  {!error && !loading && entries.length === 0 && <EmptyState />}
  <WordSheet
    wordId={selectedWordId}
    open={selectedWordId !== null}
    onOpenChange={(open) => !open && setSelectedWordId(null)}
    onOpenFullPage={(id) => navigate(`/dictionary/${id}`)}
    onEdit={(id) => { setSelectedWordId(null); setEditWordId(id) }}
    onDelete={(entry) => { setSelectedWordId(null); deleteWord(entry) }}
  />
  <WordEditDialog
    wordId={editWordId}
    open={editWordId !== null}
    onOpenChange={(open) => !open && setEditWordId(null)}
  />
</div>
```

EmptyState: conditional on whether filters are active (show "clear filters" button) or dictionary is truly empty.

**Step 2: Verify build**

Run: `cd frontend-real && npx tsc --noEmit`

**Step 3: Verify visually**

Run: `cd frontend-real && npm run dev` — open http://localhost:5173/dictionary
Check: page renders, toolbar appears, loading skeletons show, then data (or empty state).

**Step 4: Commit**

```bash
git add src/pages/DictionaryPage.tsx
git commit -m "feat(dictionary): implement DictionaryPage with search, filters, and views"
```

---

### Task 11: DictionaryEntryPage — full word page

**Files:**
- Modify: `src/pages/DictionaryEntryPage.tsx` (overwrite stub)

**Dependencies:** Task 4 (useWordDetail), Task 8 (WordOverview)

**Step 1: Implement DictionaryEntryPage**

Overwrite `src/pages/DictionaryEntryPage.tsx`.

```typescript
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowLeft, Pencil, Trash2 } from 'lucide-react'
import { useWordDetail } from '@/hooks/useWordDetail'
import { useDeleteWord } from '@/hooks/useDeleteWord'
import { WordOverview } from '@/components/dictionary/WordOverview'
import { WordEditDialog } from '@/components/dictionary/WordEditDialog'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
```

Implementation:
- `useParams()` to get `id`.
- `useWordDetail(id)` for data.
- Back button: `<Button variant="ghost">` with `<ArrowLeft>` + `t('entry.backToDict')`, navigates to `/dictionary`.
- Loading: skeleton layout matching WordOverview structure.
- Error: inline message with `t('error.loadFailed')` + `t('error.tryAgain')` button.
- Content: `<WordOverview entry={entry}>` at full width.
- Action buttons: "Edit" and "Delete" in a flex row, top-right or below back button.
- `<WordEditDialog>` controlled by local state.
- Delete navigates back to `/dictionary` after deletion.

**Step 2: Verify build**

Run: `cd frontend-real && npx tsc --noEmit`

**Step 3: Commit**

```bash
git add src/pages/DictionaryEntryPage.tsx
git commit -m "feat(dictionary): implement DictionaryEntryPage"
```

---

### Task 12: WordEditDialog + useWordEdit hook

**Files:**
- Create: `src/hooks/useWordEdit.ts`
- Create: `src/components/dictionary/WordEditDialog.tsx`

**Dependencies:** Task 1 (types), Task 2 (i18n), Task 4 (useWordDetail)

This is the most complex component. Edit modal with sections for word text, topics, notes, senses (with translations/examples), and media (images/pronunciations).

**Step 1: Create useWordEdit hook**

Create `src/hooks/useWordEdit.ts`.

This hook manages the edit form state and the UPDATE_WORD mutation.

```typescript
import { useMutation } from '@apollo/client'
import { UPDATE_WORD, GET_DICTIONARY } from '@/graphql/queries/dictionary'
import type { DictionaryEntry } from '@/types/dictionary'

interface UseWordEditReturn {
  updateWord: (id: string, input: UpdateWordInput) => Promise<void>
  loading: boolean
}

// Input types mirror the GraphQL UpdateWordInput
interface UpdateWordInput {
  text?: string
  notes?: string
  senses?: SenseInput[]
  images?: ImageInput[]
  pronunciations?: PronunciationInput[]
  topicIDs?: string[]
}

interface SenseInput {
  definition?: string
  partOfSpeech?: string
  translations: { text: string }[]
  examples?: { sentence: string; translation?: string }[]
}

interface ImageInput {
  url: string
  caption?: string
}

interface PronunciationInput {
  transcription: string
  audioUrl?: string
  region?: string
}
```

The hook calls `UPDATE_WORD` mutation and refetches `GET_DICTIONARY` on success.

**Step 2: Create WordEditDialog**

Create `src/components/dictionary/WordEditDialog.tsx`.

Props:

```typescript
interface WordEditDialogProps {
  wordId: string | null
  open: boolean
  onOpenChange: (open: boolean) => void
}
```

Implementation:
- Uses `useWordDetail(wordId)` to load current data.
- Uses `useWordEdit()` for the mutation.
- Local form state initialized from entry data when dialog opens (useEffect on entry).
- `<Dialog open={open} onOpenChange={onOpenChange}>` with `<DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto">`.
- `<DialogHeader>` with `<DialogTitle>{t('edit.title')}</DialogTitle>`.
- Form sections:
  1. **Word**: `<Input>` for text.
  2. **Topics**: Multi-select using `<DropdownMenu>` with checkboxes. Uses `GET_TOPICS` query.
  3. **Notes**: `<textarea>` with Tailwind classes matching Input styling.
  4. **Senses**: Dynamic list. Each sense has:
     - POS: `<select>` or dropdown with all PartOfSpeech values.
     - Definition: `<Input>`.
     - Translations: dynamic list of `<Input>`, add/remove buttons.
     - Examples: dynamic list, each with sentence `<Input>` + translation `<Input>`.
     - Remove sense button (`<Trash2>` icon).
     - "Add meaning" button at bottom.
  5. **Media — Images**: dynamic list, each with URL `<Input>` + caption `<Input>`.
  6. **Media — Pronunciations**: dynamic list, each with transcription `<Input>` + audioUrl `<Input>` + region `<Input>`.
- `<DialogFooter>`: Cancel (`<Button variant="outline">`) and Save (`<Button>`, disabled while loading, shows `t('edit.saving')` while submitting).
- On save: construct `UpdateWordInput` from form state, call `updateWord(id, input)`, close dialog on success, show error toast on failure.

**Step 3: Verify build**

Run: `cd frontend-real && npx tsc --noEmit`

**Step 4: Verify visually**

Run: `cd frontend-real && npm run dev`
- Open dictionary, click a word to open Sheet, click "Edit" to open dialog.
- Verify all form fields render, add/remove senses works, save sends mutation.

**Step 5: Commit**

```bash
git add src/hooks/useWordEdit.ts src/components/dictionary/WordEditDialog.tsx
git commit -m "feat(dictionary): add WordEditDialog with full edit form"
```

---

### Task 13: Final integration & polish

**Dependencies:** All previous tasks

**Step 1: Verify full flow**

Run: `cd frontend-real && npm run dev`

Test manually:
- [ ] Dictionary page loads with word list
- [ ] Grid view shows cards correctly
- [ ] Table view shows rows correctly
- [ ] View mode toggle works and persists to localStorage
- [ ] Search filters words (debounced)
- [ ] Topic filter works (multi-select)
- [ ] Part of speech filter works
- [ ] Sort by alphabetical/created/updated + ASC/DESC
- [ ] Click word → Sheet opens with full info
- [ ] Sheet "Open full page" → navigates to /dictionary/:id
- [ ] Sheet "Edit" → opens edit dialog
- [ ] Sheet "Delete" → deletes with undo toast
- [ ] DictionaryEntryPage loads correctly
- [ ] Edit dialog saves changes
- [ ] Empty state shows when no words
- [ ] Empty state shows when filters yield no results
- [ ] Loading skeletons display correctly
- [ ] Error states display correctly
- [ ] i18n works in both EN and RU

**Step 2: Run lint**

Run: `cd frontend-real && npm run lint`
Fix any linting issues.

**Step 3: Run build**

Run: `cd frontend-real && npm run build`
Fix any build errors.

**Step 4: Commit any polish fixes**

```bash
git add -A
git commit -m "fix(dictionary): polish and fix lint issues"
```
