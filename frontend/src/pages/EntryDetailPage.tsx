import { useState, useEffect, useCallback } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useGraphQL } from '../hooks/useGraphQL'
import { RawPanel } from '../components/RawPanel'

// ---------- GraphQL Queries / Mutations ----------

const DICTIONARY_ENTRY = `
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
}`

const UPDATE_NOTES = `
mutation UpdateNotes($input: UpdateEntryNotesInput!) {
  updateEntryNotes(input: $input) { entry { id notes } }
}`

const CREATE_CARD = `
mutation CreateCard($entryId: UUID!) {
  createCard(entryId: $entryId) { card { id status } }
}`

const DELETE_CARD = `
mutation DeleteCard($id: UUID!) {
  deleteCard(id: $id) { cardId }
}`

const ADD_SENSE = `
mutation AddSense($input: AddSenseInput!) {
  addSense(input: $input) { sense { id definition partOfSpeech cefrLevel position } }
}`

const UPDATE_SENSE = `
mutation UpdateSense($input: UpdateSenseInput!) {
  updateSense(input: $input) { sense { id definition partOfSpeech cefrLevel } }
}`

const DELETE_SENSE = `
mutation DeleteSense($id: UUID!) {
  deleteSense(id: $id) { senseId }
}`

const REORDER_SENSES = `
mutation ReorderSenses($input: ReorderSensesInput!) {
  reorderSenses(input: $input) { success }
}`

const ADD_TRANSLATION = `
mutation AddTranslation($input: AddTranslationInput!) {
  addTranslation(input: $input) { translation { id text position } }
}`

const UPDATE_TRANSLATION = `
mutation UpdateTranslation($input: UpdateTranslationInput!) {
  updateTranslation(input: $input) { translation { id text } }
}`

const DELETE_TRANSLATION = `
mutation DeleteTranslation($id: UUID!) {
  deleteTranslation(id: $id) { translationId }
}`

const REORDER_TRANSLATIONS = `
mutation ReorderTranslations($input: ReorderTranslationsInput!) {
  reorderTranslations(input: $input) { success }
}`

const ADD_EXAMPLE = `
mutation AddExample($input: AddExampleInput!) {
  addExample(input: $input) { example { id sentence translation position } }
}`

const UPDATE_EXAMPLE = `
mutation UpdateExample($input: UpdateExampleInput!) {
  updateExample(input: $input) { example { id sentence translation } }
}`

const DELETE_EXAMPLE = `
mutation DeleteExample($id: UUID!) {
  deleteExample(id: $id) { exampleId }
}`

const REORDER_EXAMPLES = `
mutation ReorderExamples($input: ReorderExamplesInput!) {
  reorderExamples(input: $input) { success }
}`

const ADD_USER_IMAGE = `
mutation AddUserImage($input: AddUserImageInput!) {
  addUserImage(input: $input) { image { id url caption createdAt } }
}`

const DELETE_USER_IMAGE = `
mutation DeleteUserImage($id: UUID!) {
  deleteUserImage(id: $id) { imageId }
}`

// ---------- Types ----------

interface Translation {
  id: string
  text: string
  sourceSlug: string
  position: number
}

interface Example {
  id: string
  sentence: string
  translation: string
  sourceSlug: string
  position: number
}

interface Sense {
  id: string
  definition: string
  partOfSpeech: string
  cefrLevel: string
  sourceSlug: string
  position: number
  translations: Translation[]
  examples: Example[]
}

interface Pronunciation {
  id: string
  transcription: string
  audioUrl: string
  region: string
}

interface CatalogImage {
  id: string
  url: string
  caption: string
}

interface UserImage {
  id: string
  url: string
  caption: string
  createdAt: string
}

interface Card {
  id: string
  status: string
  nextReviewAt: string
  intervalDays: number
  easeFactor: number
  createdAt: string
  updatedAt: string
}

interface Topic {
  id: string
  name: string
}

interface Entry {
  id: string
  text: string
  textNormalized: string
  notes: string | null
  createdAt: string
  updatedAt: string
  senses: Sense[]
  pronunciations: Pronunciation[]
  catalogImages: CatalogImage[]
  userImages: UserImage[]
  card: Card | null
  topics: Topic[]
}

interface DictionaryEntryData {
  dictionaryEntry: Entry
}

const PART_OF_SPEECH_OPTIONS = [
  'NOUN', 'VERB', 'ADJECTIVE', 'ADVERB', 'PRONOUN',
  'PREPOSITION', 'CONJUNCTION', 'INTERJECTION', 'PHRASE', 'IDIOM', 'OTHER',
]

// ---------- Component ----------

export function EntryDetailPage() {
  const { id } = useParams<{ id: string }>()

  // Main query
  const query = useGraphQL<DictionaryEntryData>()
  // Mutation hook (reused for all mutations)
  const mutation = useGraphQL()

  // Notes editing
  const [editingNotes, setEditingNotes] = useState(false)
  const [notesValue, setNotesValue] = useState('')

  // Sense editing
  const [editingSenseId, setEditingSenseId] = useState<string | null>(null)
  const [senseForm, setSenseForm] = useState({ definition: '', partOfSpeech: '', cefrLevel: '' })

  // Sense collapse state
  const [collapsedSenses, setCollapsedSenses] = useState<Set<string>>(new Set())

  // Add sense form
  const [showAddSense, setShowAddSense] = useState(false)
  const [addSenseForm, setAddSenseForm] = useState({
    definition: '', partOfSpeech: 'NOUN', cefrLevel: '', translations: '',
  })

  // Translation editing
  const [editingTranslationId, setEditingTranslationId] = useState<string | null>(null)
  const [translationText, setTranslationText] = useState('')

  // Add translation
  const [addTranslationSenseId, setAddTranslationSenseId] = useState<string | null>(null)
  const [addTranslationText, setAddTranslationText] = useState('')

  // Example editing
  const [editingExampleId, setEditingExampleId] = useState<string | null>(null)
  const [exampleForm, setExampleForm] = useState({ sentence: '', translation: '' })

  // Add example
  const [addExampleSenseId, setAddExampleSenseId] = useState<string | null>(null)
  const [addExampleForm, setAddExampleForm] = useState({ sentence: '', translation: '' })

  // User image form
  const [showAddImage, setShowAddImage] = useState(false)
  const [imageForm, setImageForm] = useState({ url: '', caption: '' })

  // Track the last raw for RawPanel
  const lastRaw = mutation.raw ?? query.raw

  // ---------- Data loading ----------

  const refetch = useCallback(async () => {
    if (!id) return
    await query.execute(DICTIONARY_ENTRY, { id })
  }, [id, query.execute])

  useEffect(() => {
    refetch()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id])

  const entry = query.data?.dictionaryEntry ?? null

  // Helper to run a mutation and then refetch
  async function mutateAndRefetch(
    gql: string,
    variables: Record<string, unknown>,
  ) {
    const data = await mutation.execute(gql, variables)
    if (data) {
      await refetch()
    }
    return data
  }

  // ---------- Notes ----------

  function startEditNotes() {
    setEditingNotes(true)
    setNotesValue(entry?.notes ?? '')
  }

  async function saveNotes() {
    if (!entry) return
    await mutateAndRefetch(UPDATE_NOTES, {
      input: { entryId: entry.id, notes: notesValue || null },
    })
    setEditingNotes(false)
  }

  // ---------- Card ----------

  async function handleCreateCard() {
    if (!entry) return
    await mutateAndRefetch(CREATE_CARD, { entryId: entry.id })
  }

  async function handleDeleteCard() {
    if (!entry?.card) return
    await mutateAndRefetch(DELETE_CARD, { id: entry.card.id })
  }

  // ---------- Senses ----------

  function startEditSense(sense: Sense) {
    setEditingSenseId(sense.id)
    setSenseForm({
      definition: sense.definition,
      partOfSpeech: sense.partOfSpeech,
      cefrLevel: sense.cefrLevel ?? '',
    })
  }

  async function saveEditSense() {
    if (!editingSenseId) return
    await mutateAndRefetch(UPDATE_SENSE, {
      input: {
        senseId: editingSenseId,
        definition: senseForm.definition || null,
        partOfSpeech: senseForm.partOfSpeech || null,
        cefrLevel: senseForm.cefrLevel || null,
      },
    })
    setEditingSenseId(null)
  }

  async function handleDeleteSense(senseId: string) {
    await mutateAndRefetch(DELETE_SENSE, { id: senseId })
  }

  async function handleMoveSense(senseId: string, direction: 'up' | 'down') {
    if (!entry) return
    const sorted = [...entry.senses].sort((a, b) => a.position - b.position)
    const idx = sorted.findIndex((s) => s.id === senseId)
    if (idx < 0) return
    if (direction === 'up' && idx === 0) return
    if (direction === 'down' && idx === sorted.length - 1) return

    const swapIdx = direction === 'up' ? idx - 1 : idx + 1
    const items = sorted.map((s, i) => {
      if (i === idx) return { id: s.id, position: sorted[swapIdx].position }
      if (i === swapIdx) return { id: s.id, position: sorted[idx].position }
      return { id: s.id, position: s.position }
    })

    await mutateAndRefetch(REORDER_SENSES, {
      input: { entryId: entry.id, items },
    })
  }

  async function handleAddSense(e: React.FormEvent) {
    e.preventDefault()
    if (!entry) return
    const translations = addSenseForm.translations
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean)

    await mutateAndRefetch(ADD_SENSE, {
      input: {
        entryId: entry.id,
        definition: addSenseForm.definition || null,
        partOfSpeech: addSenseForm.partOfSpeech || null,
        cefrLevel: addSenseForm.cefrLevel || null,
        translations: translations.length > 0 ? translations : null,
      },
    })
    setAddSenseForm({ definition: '', partOfSpeech: 'NOUN', cefrLevel: '', translations: '' })
    setShowAddSense(false)
  }

  // ---------- Translations ----------

  function startEditTranslation(t: Translation) {
    setEditingTranslationId(t.id)
    setTranslationText(t.text)
  }

  async function saveEditTranslation() {
    if (!editingTranslationId) return
    await mutateAndRefetch(UPDATE_TRANSLATION, {
      input: { translationId: editingTranslationId, text: translationText },
    })
    setEditingTranslationId(null)
  }

  async function handleDeleteTranslation(translationId: string) {
    await mutateAndRefetch(DELETE_TRANSLATION, { id: translationId })
  }

  async function handleAddTranslation(senseId: string) {
    if (!addTranslationText.trim()) return
    await mutateAndRefetch(ADD_TRANSLATION, {
      input: { senseId, text: addTranslationText.trim() },
    })
    setAddTranslationSenseId(null)
    setAddTranslationText('')
  }

  async function handleMoveTranslation(senseId: string, translationId: string, direction: 'up' | 'down') {
    const sense = entry?.senses.find((s) => s.id === senseId)
    if (!sense) return
    const sorted = [...sense.translations].sort((a, b) => a.position - b.position)
    const idx = sorted.findIndex((t) => t.id === translationId)
    if (idx < 0) return
    if (direction === 'up' && idx === 0) return
    if (direction === 'down' && idx === sorted.length - 1) return

    const swapIdx = direction === 'up' ? idx - 1 : idx + 1
    const items = sorted.map((t, i) => {
      if (i === idx) return { id: t.id, position: sorted[swapIdx].position }
      if (i === swapIdx) return { id: t.id, position: sorted[idx].position }
      return { id: t.id, position: t.position }
    })

    await mutateAndRefetch(REORDER_TRANSLATIONS, {
      input: { senseId, items },
    })
  }

  // ---------- Examples ----------

  function startEditExample(ex: Example) {
    setEditingExampleId(ex.id)
    setExampleForm({ sentence: ex.sentence, translation: ex.translation ?? '' })
  }

  async function saveEditExample() {
    if (!editingExampleId) return
    await mutateAndRefetch(UPDATE_EXAMPLE, {
      input: {
        exampleId: editingExampleId,
        sentence: exampleForm.sentence || null,
        translation: exampleForm.translation || null,
      },
    })
    setEditingExampleId(null)
  }

  async function handleDeleteExample(exampleId: string) {
    await mutateAndRefetch(DELETE_EXAMPLE, { id: exampleId })
  }

  async function handleAddExample(senseId: string) {
    if (!addExampleForm.sentence.trim()) return
    await mutateAndRefetch(ADD_EXAMPLE, {
      input: {
        senseId,
        sentence: addExampleForm.sentence.trim(),
        translation: addExampleForm.translation.trim() || null,
      },
    })
    setAddExampleSenseId(null)
    setAddExampleForm({ sentence: '', translation: '' })
  }

  async function handleMoveExample(senseId: string, exampleId: string, direction: 'up' | 'down') {
    const sense = entry?.senses.find((s) => s.id === senseId)
    if (!sense) return
    const sorted = [...sense.examples].sort((a, b) => a.position - b.position)
    const idx = sorted.findIndex((ex) => ex.id === exampleId)
    if (idx < 0) return
    if (direction === 'up' && idx === 0) return
    if (direction === 'down' && idx === sorted.length - 1) return

    const swapIdx = direction === 'up' ? idx - 1 : idx + 1
    const items = sorted.map((ex, i) => {
      if (i === idx) return { id: ex.id, position: sorted[swapIdx].position }
      if (i === swapIdx) return { id: ex.id, position: sorted[idx].position }
      return { id: ex.id, position: ex.position }
    })

    await mutateAndRefetch(REORDER_EXAMPLES, {
      input: { senseId, items },
    })
  }

  // ---------- User Images ----------

  async function handleAddUserImage(e: React.FormEvent) {
    e.preventDefault()
    if (!entry || !imageForm.url.trim()) return
    await mutateAndRefetch(ADD_USER_IMAGE, {
      input: {
        entryId: entry.id,
        url: imageForm.url.trim(),
        caption: imageForm.caption.trim() || null,
      },
    })
    setImageForm({ url: '', caption: '' })
    setShowAddImage(false)
  }

  async function handleDeleteUserImage(imageId: string) {
    await mutateAndRefetch(DELETE_USER_IMAGE, { id: imageId })
  }

  // ---------- Sense collapse ----------

  function toggleSenseCollapse(senseId: string) {
    setCollapsedSenses((prev) => {
      const next = new Set(prev)
      if (next.has(senseId)) {
        next.delete(senseId)
      } else {
        next.add(senseId)
      }
      return next
    })
  }

  // ---------- Render ----------

  if (!id) {
    return <div className="p-6 text-red-500">No entry ID provided.</div>
  }

  if (query.loading && !entry) {
    return <div className="p-6 text-gray-500">Loading entry...</div>
  }

  if (query.errors && !entry) {
    return (
      <div className="p-6">
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
          {query.errors.map((err, i) => (
            <div key={i}>{err.message}</div>
          ))}
        </div>
        <Link to="/dictionary" className="text-blue-600 text-sm hover:underline mt-4 inline-block">
          Back to Dictionary
        </Link>
        <RawPanel raw={lastRaw} />
      </div>
    )
  }

  if (!entry) {
    return (
      <div className="p-6 text-gray-500">
        Entry not found.
        <Link to="/dictionary" className="text-blue-600 text-sm hover:underline ml-2">
          Back to Dictionary
        </Link>
        <RawPanel raw={lastRaw} />
      </div>
    )
  }

  const sortedSenses = [...entry.senses].sort((a, b) => a.position - b.position)

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      {/* Back link */}
      <Link to="/dictionary" className="text-blue-600 text-sm hover:underline">
        &larr; Back to Dictionary
      </Link>

      {/* Mutation errors */}
      {mutation.errors && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
          {mutation.errors.map((err, i) => (
            <div key={i}>{err.message}</div>
          ))}
        </div>
      )}

      {/* Mutation loading indicator */}
      {mutation.loading && (
        <div className="text-sm text-blue-600">Saving...</div>
      )}

      {/* ===== 1. Entry Header ===== */}
      <section className="bg-white border border-gray-200 rounded-lg p-5 shadow-sm">
        <h1 className="text-3xl font-bold text-gray-900 mb-1">{entry.text}</h1>
        {entry.textNormalized !== entry.text && (
          <div className="text-sm text-gray-400 mb-2">Normalized: {entry.textNormalized}</div>
        )}

        {/* Topics */}
        {entry.topics.length > 0 && (
          <div className="flex flex-wrap gap-1 mb-3">
            {entry.topics.map((topic) => (
              <span
                key={topic.id}
                className="text-xs bg-indigo-100 text-indigo-700 px-2 py-0.5 rounded-full"
              >
                {topic.name}
              </span>
            ))}
          </div>
        )}

        {/* Dates */}
        <div className="text-xs text-gray-400 mb-3">
          Created: {new Date(entry.createdAt).toLocaleString()} | Updated: {new Date(entry.updatedAt).toLocaleString()}
        </div>

        {/* Notes */}
        <div className="mt-2">
          <h3 className="text-sm font-semibold text-gray-700 mb-1">Notes</h3>
          {editingNotes ? (
            <div className="space-y-2">
              <textarea
                value={notesValue}
                onChange={(e) => setNotesValue(e.target.value)}
                rows={4}
                className="w-full border border-gray-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
                placeholder="Add personal notes..."
              />
              <div className="flex gap-2">
                <button
                  onClick={saveNotes}
                  disabled={mutation.loading}
                  className="text-xs px-3 py-1 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
                >
                  Save Notes
                </button>
                <button
                  onClick={() => setEditingNotes(false)}
                  className="text-xs px-3 py-1 bg-gray-200 text-gray-700 rounded hover:bg-gray-300"
                >
                  Cancel
                </button>
              </div>
            </div>
          ) : (
            <div className="flex items-start gap-2">
              <div className="text-sm text-gray-600 flex-1 whitespace-pre-wrap">
                {entry.notes || <span className="italic text-gray-400">No notes</span>}
              </div>
              <button
                onClick={startEditNotes}
                className="text-xs px-2 py-1 bg-gray-100 text-gray-600 rounded hover:bg-gray-200 shrink-0"
              >
                Edit
              </button>
            </div>
          )}
        </div>
      </section>

      {/* ===== 2. Card Section ===== */}
      <section className="bg-white border border-gray-200 rounded-lg p-5 shadow-sm">
        <h2 className="text-lg font-bold text-gray-800 mb-3">Study Card</h2>
        {entry.card ? (
          <div>
            <div className="grid grid-cols-2 gap-2 text-sm mb-3">
              <div>
                <span className="text-gray-500">Status:</span>{' '}
                <span className="font-medium">{entry.card.status}</span>
              </div>
              <div>
                <span className="text-gray-500">Next Review:</span>{' '}
                <span className="font-medium">
                  {entry.card.nextReviewAt
                    ? new Date(entry.card.nextReviewAt).toLocaleString()
                    : 'N/A'}
                </span>
              </div>
              <div>
                <span className="text-gray-500">Interval:</span>{' '}
                <span className="font-medium">{entry.card.intervalDays} days</span>
              </div>
              <div>
                <span className="text-gray-500">Ease Factor:</span>{' '}
                <span className="font-medium">{entry.card.easeFactor}</span>
              </div>
              <div>
                <span className="text-gray-500">Created:</span>{' '}
                <span className="text-xs">{new Date(entry.card.createdAt).toLocaleString()}</span>
              </div>
              <div>
                <span className="text-gray-500">Updated:</span>{' '}
                <span className="text-xs">{new Date(entry.card.updatedAt).toLocaleString()}</span>
              </div>
            </div>
            <button
              onClick={handleDeleteCard}
              disabled={mutation.loading}
              className="text-xs px-3 py-1 bg-red-50 text-red-700 border border-red-200 rounded hover:bg-red-100 disabled:opacity-50"
            >
              Delete Card
            </button>
          </div>
        ) : (
          <div>
            <p className="text-sm text-gray-500 mb-2">No study card exists for this entry.</p>
            <button
              onClick={handleCreateCard}
              disabled={mutation.loading}
              className="text-xs px-3 py-1 bg-green-50 text-green-700 border border-green-200 rounded hover:bg-green-100 disabled:opacity-50"
            >
              Create Card
            </button>
          </div>
        )}
      </section>

      {/* ===== 3. Pronunciations (read-only) ===== */}
      {entry.pronunciations.length > 0 && (
        <section className="bg-white border border-gray-200 rounded-lg p-5 shadow-sm">
          <h2 className="text-lg font-bold text-gray-800 mb-3">Pronunciations</h2>
          <div className="space-y-2">
            {entry.pronunciations.map((p) => (
              <div key={p.id} className="flex items-center gap-3 text-sm">
                {p.region && (
                  <span className="text-xs bg-gray-100 text-gray-600 px-2 py-0.5 rounded">
                    {p.region}
                  </span>
                )}
                <span className="font-mono text-gray-700">{p.transcription}</span>
                {p.audioUrl && (
                  <a
                    href={p.audioUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-500 underline text-xs"
                  >
                    audio
                  </a>
                )}
              </div>
            ))}
          </div>
        </section>
      )}

      {/* ===== 4. Catalog Images (read-only) ===== */}
      {entry.catalogImages.length > 0 && (
        <section className="bg-white border border-gray-200 rounded-lg p-5 shadow-sm">
          <h2 className="text-lg font-bold text-gray-800 mb-3">Catalog Images</h2>
          <div className="flex flex-wrap gap-3">
            {entry.catalogImages.map((img) => (
              <div key={img.id} className="text-center">
                <img
                  src={img.url}
                  alt={img.caption || 'catalog image'}
                  className="h-32 rounded border border-gray-200"
                />
                {img.caption && (
                  <div className="text-xs text-gray-500 mt-1">{img.caption}</div>
                )}
              </div>
            ))}
          </div>
        </section>
      )}

      {/* ===== 5. User Images ===== */}
      <section className="bg-white border border-gray-200 rounded-lg p-5 shadow-sm">
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-lg font-bold text-gray-800">User Images</h2>
          <button
            onClick={() => setShowAddImage(!showAddImage)}
            className="text-xs px-3 py-1 bg-green-50 text-green-700 border border-green-200 rounded hover:bg-green-100"
          >
            {showAddImage ? 'Cancel' : 'Add Image'}
          </button>
        </div>

        {showAddImage && (
          <form onSubmit={handleAddUserImage} className="mb-4 p-3 bg-gray-50 rounded border border-gray-200 space-y-2">
            <input
              type="text"
              value={imageForm.url}
              onChange={(e) => setImageForm({ ...imageForm, url: e.target.value })}
              placeholder="Image URL"
              className="w-full border border-gray-300 rounded px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-green-400"
            />
            <input
              type="text"
              value={imageForm.caption}
              onChange={(e) => setImageForm({ ...imageForm, caption: e.target.value })}
              placeholder="Caption (optional)"
              className="w-full border border-gray-300 rounded px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-green-400"
            />
            <button
              type="submit"
              disabled={mutation.loading || !imageForm.url.trim()}
              className="text-xs px-3 py-1 bg-green-600 text-white rounded hover:bg-green-700 disabled:opacity-50"
            >
              Add Image
            </button>
          </form>
        )}

        {entry.userImages.length > 0 ? (
          <div className="flex flex-wrap gap-3">
            {entry.userImages.map((img) => (
              <div key={img.id} className="text-center relative group">
                <img
                  src={img.url}
                  alt={img.caption || 'user image'}
                  className="h-32 rounded border border-gray-200"
                />
                {img.caption && (
                  <div className="text-xs text-gray-500 mt-1">{img.caption}</div>
                )}
                <div className="text-xs text-gray-400 mt-0.5">
                  {new Date(img.createdAt).toLocaleDateString()}
                </div>
                <button
                  onClick={() => handleDeleteUserImage(img.id)}
                  disabled={mutation.loading}
                  className="mt-1 text-xs px-2 py-0.5 bg-red-50 text-red-600 border border-red-200 rounded hover:bg-red-100 disabled:opacity-50"
                >
                  Delete
                </button>
              </div>
            ))}
          </div>
        ) : (
          <p className="text-sm text-gray-400">No user images.</p>
        )}
      </section>

      {/* ===== 6. Senses (main content area) ===== */}
      <section className="space-y-4">
        <h2 className="text-lg font-bold text-gray-800">Senses ({sortedSenses.length})</h2>

        {sortedSenses.map((sense, senseIdx) => {
          const isCollapsed = collapsedSenses.has(sense.id)
          const isEditing = editingSenseId === sense.id
          const sortedTranslations = [...sense.translations].sort((a, b) => a.position - b.position)
          const sortedExamples = [...sense.examples].sort((a, b) => a.position - b.position)

          return (
            <div key={sense.id} className="bg-white border border-gray-200 rounded-lg shadow-sm">
              {/* Sense header */}
              <div
                className="flex items-center justify-between p-4 cursor-pointer hover:bg-gray-50"
                onClick={() => toggleSenseCollapse(sense.id)}
              >
                <div className="flex items-center gap-2 flex-1 min-w-0">
                  <span className="text-gray-400 text-xs shrink-0">{isCollapsed ? '>' : 'v'}</span>
                  <span className="font-medium text-indigo-700 text-sm shrink-0">
                    {sense.partOfSpeech}
                  </span>
                  {sense.cefrLevel && (
                    <span className="text-xs bg-yellow-100 text-yellow-800 px-1.5 py-0.5 rounded shrink-0">
                      {sense.cefrLevel}
                    </span>
                  )}
                  <span className="text-sm text-gray-700 truncate">
                    {sense.definition || '(no definition)'}
                  </span>
                  {sense.sourceSlug && (
                    <span className="text-xs text-gray-400 shrink-0">[{sense.sourceSlug}]</span>
                  )}
                </div>
                <div className="flex items-center gap-1 ml-2 shrink-0" onClick={(e) => e.stopPropagation()}>
                  <button
                    onClick={() => handleMoveSense(sense.id, 'up')}
                    disabled={mutation.loading || senseIdx === 0}
                    className="text-xs px-1.5 py-0.5 bg-gray-100 text-gray-600 rounded hover:bg-gray-200 disabled:opacity-30"
                    title="Move Up"
                  >
                    Up
                  </button>
                  <button
                    onClick={() => handleMoveSense(sense.id, 'down')}
                    disabled={mutation.loading || senseIdx === sortedSenses.length - 1}
                    className="text-xs px-1.5 py-0.5 bg-gray-100 text-gray-600 rounded hover:bg-gray-200 disabled:opacity-30"
                    title="Move Down"
                  >
                    Down
                  </button>
                  <button
                    onClick={() => startEditSense(sense)}
                    className="text-xs px-2 py-0.5 bg-blue-50 text-blue-700 rounded hover:bg-blue-100"
                  >
                    Edit
                  </button>
                  <button
                    onClick={() => handleDeleteSense(sense.id)}
                    disabled={mutation.loading}
                    className="text-xs px-2 py-0.5 bg-red-50 text-red-600 rounded hover:bg-red-100 disabled:opacity-50"
                  >
                    Delete
                  </button>
                </div>
              </div>

              {/* Sense edit form (inline) */}
              {isEditing && (
                <div className="px-4 pb-3 border-t border-gray-100 pt-3 bg-blue-50">
                  <div className="grid grid-cols-3 gap-2 mb-2">
                    <input
                      type="text"
                      value={senseForm.definition}
                      onChange={(e) => setSenseForm({ ...senseForm, definition: e.target.value })}
                      placeholder="Definition"
                      className="border border-gray-300 rounded px-2 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
                    />
                    <select
                      value={senseForm.partOfSpeech}
                      onChange={(e) => setSenseForm({ ...senseForm, partOfSpeech: e.target.value })}
                      className="border border-gray-300 rounded px-2 py-1 text-sm bg-white"
                    >
                      <option value="">-- Part of Speech --</option>
                      {PART_OF_SPEECH_OPTIONS.map((pos) => (
                        <option key={pos} value={pos}>{pos}</option>
                      ))}
                    </select>
                    <input
                      type="text"
                      value={senseForm.cefrLevel}
                      onChange={(e) => setSenseForm({ ...senseForm, cefrLevel: e.target.value })}
                      placeholder="CEFR Level"
                      className="border border-gray-300 rounded px-2 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
                    />
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={saveEditSense}
                      disabled={mutation.loading}
                      className="text-xs px-3 py-1 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
                    >
                      Save
                    </button>
                    <button
                      onClick={() => setEditingSenseId(null)}
                      className="text-xs px-3 py-1 bg-gray-200 text-gray-700 rounded hover:bg-gray-300"
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              )}

              {/* Sense body (collapsible) */}
              {!isCollapsed && (
                <div className="px-4 pb-4 border-t border-gray-100 space-y-4 pt-3">
                  {/* Translations */}
                  <div>
                    <div className="flex items-center justify-between mb-2">
                      <h4 className="text-sm font-semibold text-gray-600">
                        Translations ({sortedTranslations.length})
                      </h4>
                      <button
                        onClick={() => {
                          setAddTranslationSenseId(
                            addTranslationSenseId === sense.id ? null : sense.id,
                          )
                          setAddTranslationText('')
                        }}
                        className="text-xs px-2 py-0.5 bg-green-50 text-green-700 border border-green-200 rounded hover:bg-green-100"
                      >
                        {addTranslationSenseId === sense.id ? 'Cancel' : 'Add Translation'}
                      </button>
                    </div>

                    {addTranslationSenseId === sense.id && (
                      <div className="flex gap-2 mb-2">
                        <input
                          type="text"
                          value={addTranslationText}
                          onChange={(e) => setAddTranslationText(e.target.value)}
                          placeholder="Translation text"
                          className="flex-1 border border-gray-300 rounded px-2 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-green-400"
                          onKeyDown={(e) => {
                            if (e.key === 'Enter') {
                              e.preventDefault()
                              handleAddTranslation(sense.id)
                            }
                          }}
                        />
                        <button
                          onClick={() => handleAddTranslation(sense.id)}
                          disabled={mutation.loading || !addTranslationText.trim()}
                          className="text-xs px-3 py-1 bg-green-600 text-white rounded hover:bg-green-700 disabled:opacity-50"
                        >
                          Add
                        </button>
                      </div>
                    )}

                    {sortedTranslations.length > 0 ? (
                      <div className="space-y-1">
                        {sortedTranslations.map((t, tIdx) => (
                          <div key={t.id} className="flex items-center gap-2 text-sm group">
                            {editingTranslationId === t.id ? (
                              <>
                                <input
                                  type="text"
                                  value={translationText}
                                  onChange={(e) => setTranslationText(e.target.value)}
                                  className="flex-1 border border-gray-300 rounded px-2 py-0.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
                                  onKeyDown={(e) => {
                                    if (e.key === 'Enter') {
                                      e.preventDefault()
                                      saveEditTranslation()
                                    }
                                    if (e.key === 'Escape') setEditingTranslationId(null)
                                  }}
                                />
                                <button
                                  onClick={saveEditTranslation}
                                  disabled={mutation.loading}
                                  className="text-xs px-2 py-0.5 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
                                >
                                  Save
                                </button>
                                <button
                                  onClick={() => setEditingTranslationId(null)}
                                  className="text-xs px-2 py-0.5 bg-gray-200 text-gray-700 rounded hover:bg-gray-300"
                                >
                                  Cancel
                                </button>
                              </>
                            ) : (
                              <>
                                <span className="flex-1 text-gray-700">
                                  {t.text}
                                  {t.sourceSlug && (
                                    <span className="text-xs text-gray-400 ml-1">[{t.sourceSlug}]</span>
                                  )}
                                </span>
                                <div className="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                                  <button
                                    onClick={() => handleMoveTranslation(sense.id, t.id, 'up')}
                                    disabled={mutation.loading || tIdx === 0}
                                    className="text-xs px-1 py-0.5 bg-gray-100 text-gray-500 rounded hover:bg-gray-200 disabled:opacity-30"
                                  >
                                    Up
                                  </button>
                                  <button
                                    onClick={() => handleMoveTranslation(sense.id, t.id, 'down')}
                                    disabled={mutation.loading || tIdx === sortedTranslations.length - 1}
                                    className="text-xs px-1 py-0.5 bg-gray-100 text-gray-500 rounded hover:bg-gray-200 disabled:opacity-30"
                                  >
                                    Down
                                  </button>
                                  <button
                                    onClick={() => startEditTranslation(t)}
                                    className="text-xs px-1.5 py-0.5 bg-blue-50 text-blue-600 rounded hover:bg-blue-100"
                                  >
                                    Edit
                                  </button>
                                  <button
                                    onClick={() => handleDeleteTranslation(t.id)}
                                    disabled={mutation.loading}
                                    className="text-xs px-1.5 py-0.5 bg-red-50 text-red-600 rounded hover:bg-red-100 disabled:opacity-50"
                                  >
                                    Delete
                                  </button>
                                </div>
                              </>
                            )}
                          </div>
                        ))}
                      </div>
                    ) : (
                      <p className="text-xs text-gray-400 italic">No translations.</p>
                    )}
                  </div>

                  {/* Examples */}
                  <div>
                    <div className="flex items-center justify-between mb-2">
                      <h4 className="text-sm font-semibold text-gray-600">
                        Examples ({sortedExamples.length})
                      </h4>
                      <button
                        onClick={() => {
                          setAddExampleSenseId(
                            addExampleSenseId === sense.id ? null : sense.id,
                          )
                          setAddExampleForm({ sentence: '', translation: '' })
                        }}
                        className="text-xs px-2 py-0.5 bg-green-50 text-green-700 border border-green-200 rounded hover:bg-green-100"
                      >
                        {addExampleSenseId === sense.id ? 'Cancel' : 'Add Example'}
                      </button>
                    </div>

                    {addExampleSenseId === sense.id && (
                      <div className="flex gap-2 mb-2">
                        <input
                          type="text"
                          value={addExampleForm.sentence}
                          onChange={(e) => setAddExampleForm({ ...addExampleForm, sentence: e.target.value })}
                          placeholder="Sentence"
                          className="flex-1 border border-gray-300 rounded px-2 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-green-400"
                        />
                        <input
                          type="text"
                          value={addExampleForm.translation}
                          onChange={(e) => setAddExampleForm({ ...addExampleForm, translation: e.target.value })}
                          placeholder="Translation (optional)"
                          className="flex-1 border border-gray-300 rounded px-2 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-green-400"
                        />
                        <button
                          onClick={() => handleAddExample(sense.id)}
                          disabled={mutation.loading || !addExampleForm.sentence.trim()}
                          className="text-xs px-3 py-1 bg-green-600 text-white rounded hover:bg-green-700 disabled:opacity-50"
                        >
                          Add
                        </button>
                      </div>
                    )}

                    {sortedExamples.length > 0 ? (
                      <div className="space-y-1">
                        {sortedExamples.map((ex, exIdx) => (
                          <div key={ex.id} className="text-sm group">
                            {editingExampleId === ex.id ? (
                              <div className="flex gap-2 items-center">
                                <input
                                  type="text"
                                  value={exampleForm.sentence}
                                  onChange={(e) => setExampleForm({ ...exampleForm, sentence: e.target.value })}
                                  placeholder="Sentence"
                                  className="flex-1 border border-gray-300 rounded px-2 py-0.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
                                />
                                <input
                                  type="text"
                                  value={exampleForm.translation}
                                  onChange={(e) => setExampleForm({ ...exampleForm, translation: e.target.value })}
                                  placeholder="Translation"
                                  className="flex-1 border border-gray-300 rounded px-2 py-0.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
                                />
                                <button
                                  onClick={saveEditExample}
                                  disabled={mutation.loading}
                                  className="text-xs px-2 py-0.5 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
                                >
                                  Save
                                </button>
                                <button
                                  onClick={() => setEditingExampleId(null)}
                                  className="text-xs px-2 py-0.5 bg-gray-200 text-gray-700 rounded hover:bg-gray-300"
                                >
                                  Cancel
                                </button>
                              </div>
                            ) : (
                              <div className="flex items-center gap-2">
                                <div className="flex-1">
                                  <span className="text-gray-700 italic">"{ex.sentence}"</span>
                                  {ex.translation && (
                                    <span className="text-gray-500 ml-2">-- {ex.translation}</span>
                                  )}
                                  {ex.sourceSlug && (
                                    <span className="text-xs text-gray-400 ml-1">[{ex.sourceSlug}]</span>
                                  )}
                                </div>
                                <div className="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                                  <button
                                    onClick={() => handleMoveExample(sense.id, ex.id, 'up')}
                                    disabled={mutation.loading || exIdx === 0}
                                    className="text-xs px-1 py-0.5 bg-gray-100 text-gray-500 rounded hover:bg-gray-200 disabled:opacity-30"
                                  >
                                    Up
                                  </button>
                                  <button
                                    onClick={() => handleMoveExample(sense.id, ex.id, 'down')}
                                    disabled={mutation.loading || exIdx === sortedExamples.length - 1}
                                    className="text-xs px-1 py-0.5 bg-gray-100 text-gray-500 rounded hover:bg-gray-200 disabled:opacity-30"
                                  >
                                    Down
                                  </button>
                                  <button
                                    onClick={() => startEditExample(ex)}
                                    className="text-xs px-1.5 py-0.5 bg-blue-50 text-blue-600 rounded hover:bg-blue-100"
                                  >
                                    Edit
                                  </button>
                                  <button
                                    onClick={() => handleDeleteExample(ex.id)}
                                    disabled={mutation.loading}
                                    className="text-xs px-1.5 py-0.5 bg-red-50 text-red-600 rounded hover:bg-red-100 disabled:opacity-50"
                                  >
                                    Delete
                                  </button>
                                </div>
                              </div>
                            )}
                          </div>
                        ))}
                      </div>
                    ) : (
                      <p className="text-xs text-gray-400 italic">No examples.</p>
                    )}
                  </div>
                </div>
              )}
            </div>
          )
        })}

        {/* ===== 7. Add Sense form ===== */}
        <div>
          {showAddSense ? (
            <form onSubmit={handleAddSense} className="bg-white border-2 border-green-300 rounded-lg p-4 space-y-3">
              <h3 className="text-sm font-bold text-green-800">Add New Sense</h3>
              <div className="grid grid-cols-3 gap-2">
                <input
                  type="text"
                  value={addSenseForm.definition}
                  onChange={(e) => setAddSenseForm({ ...addSenseForm, definition: e.target.value })}
                  placeholder="Definition"
                  className="border border-gray-300 rounded px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-green-400"
                />
                <select
                  value={addSenseForm.partOfSpeech}
                  onChange={(e) => setAddSenseForm({ ...addSenseForm, partOfSpeech: e.target.value })}
                  className="border border-gray-300 rounded px-2 py-1.5 text-sm bg-white"
                >
                  {PART_OF_SPEECH_OPTIONS.map((pos) => (
                    <option key={pos} value={pos}>{pos}</option>
                  ))}
                </select>
                <input
                  type="text"
                  value={addSenseForm.cefrLevel}
                  onChange={(e) => setAddSenseForm({ ...addSenseForm, cefrLevel: e.target.value })}
                  placeholder="CEFR Level (e.g. B2)"
                  className="border border-gray-300 rounded px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-green-400"
                />
              </div>
              <input
                type="text"
                value={addSenseForm.translations}
                onChange={(e) => setAddSenseForm({ ...addSenseForm, translations: e.target.value })}
                placeholder="Translations (comma-separated)"
                className="w-full border border-gray-300 rounded px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-green-400"
              />
              <div className="flex gap-2">
                <button
                  type="submit"
                  disabled={mutation.loading}
                  className="text-xs px-3 py-1 bg-green-600 text-white rounded hover:bg-green-700 disabled:opacity-50"
                >
                  Add Sense
                </button>
                <button
                  type="button"
                  onClick={() => setShowAddSense(false)}
                  className="text-xs px-3 py-1 bg-gray-200 text-gray-700 rounded hover:bg-gray-300"
                >
                  Cancel
                </button>
              </div>
            </form>
          ) : (
            <button
              onClick={() => setShowAddSense(true)}
              className="text-sm px-4 py-2 bg-green-50 text-green-700 border border-green-200 rounded hover:bg-green-100"
            >
              + Add Sense
            </button>
          )}
        </div>
      </section>

      {/* ===== 8. RawPanel ===== */}
      <RawPanel raw={lastRaw} />
    </div>
  )
}
