import { useState } from 'react'
import { useGraphQL } from '../hooks/useGraphQL'
import { useAuth } from '../auth/useAuth'
import { RawPanel } from '../components/RawPanel'

// ---------- GraphQL queries / mutations ----------

const SEARCH_CATALOG = `
query SearchCatalog($query: String!, $limit: Int) {
  searchCatalog(query: $query, limit: $limit) {
    id text textNormalized
    senses {
      id definition partOfSpeech cefrLevel position
      translations { id text }
      examples { id sentence translation }
    }
    pronunciations { id transcription audioUrl region }
    images { id url caption }
  }
}`

const PREVIEW_REF_ENTRY = `
query PreviewRefEntry($text: String!) {
  previewRefEntry(text: $text) {
    id text textNormalized
    senses {
      id definition partOfSpeech cefrLevel position
      translations { id text }
      examples { id sentence translation }
    }
    pronunciations { id transcription audioUrl region }
    images { id url caption }
  }
}`

const CREATE_ENTRY_FROM_CATALOG = `
mutation CreateEntryFromCatalog($input: CreateEntryFromCatalogInput!) {
  createEntryFromCatalog(input: $input) {
    entry { id text }
  }
}`

// ---------- Types ----------

interface Translation {
  id: string
  text: string
}

interface Example {
  id: string
  sentence: string
  translation: string
}

interface Sense {
  id: string
  definition: string
  partOfSpeech: string
  cefrLevel: string
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

interface Image {
  id: string
  url: string
  caption: string
}

interface RefEntry {
  id: string
  text: string
  textNormalized: string
  senses: Sense[]
  pronunciations: Pronunciation[]
  images: Image[]
}

interface SearchCatalogData {
  searchCatalog: RefEntry[]
}

interface PreviewRefEntryData {
  previewRefEntry: RefEntry
}

interface CreateEntryFromCatalogData {
  createEntryFromCatalog: {
    entry: { id: string; text: string }
  }
}

interface AddForm {
  refEntryId: string
  refEntryText: string
  senses: Sense[]
  selectedSenseIds: string[]
  notes: string
  createCard: boolean
}

// ---------- Component ----------

export function CatalogPage() {
  const { isAuthenticated } = useAuth()

  // Search state
  const [searchQuery, setSearchQuery] = useState('')
  const [limit, setLimit] = useState<number>(10)
  const search = useGraphQL<SearchCatalogData>()

  // Preview state
  const [previewEntry, setPreviewEntry] = useState<RefEntry | null>(null)
  const preview = useGraphQL<PreviewRefEntryData>()

  // Add to dictionary state
  const [addForm, setAddForm] = useState<AddForm | null>(null)
  const addEntry = useGraphQL<CreateEntryFromCatalogData>()
  const [addSuccess, setAddSuccess] = useState<string | null>(null)

  // The raw panel shows the most recent operation
  const lastRaw = addEntry.raw ?? preview.raw ?? search.raw

  // ---------- Handlers ----------

  async function handleSearch(e: React.FormEvent) {
    e.preventDefault()
    if (!searchQuery.trim()) return
    setPreviewEntry(null)
    setAddForm(null)
    setAddSuccess(null)
    await search.execute(SEARCH_CATALOG, { query: searchQuery.trim(), limit })
  }

  async function handlePreview(text: string) {
    setAddForm(null)
    setAddSuccess(null)
    const data = await preview.execute(PREVIEW_REF_ENTRY, { text })
    if (data?.previewRefEntry) {
      setPreviewEntry(data.previewRefEntry)
    }
  }

  function handleStartAdd(entry: RefEntry) {
    setPreviewEntry(null)
    setAddSuccess(null)
    setAddForm({
      refEntryId: entry.id,
      refEntryText: entry.text,
      senses: entry.senses,
      selectedSenseIds: entry.senses.map((s) => s.id),
      notes: '',
      createCard: true,
    })
  }

  function toggleSense(senseId: string) {
    if (!addForm) return
    setAddForm({
      ...addForm,
      selectedSenseIds: addForm.selectedSenseIds.includes(senseId)
        ? addForm.selectedSenseIds.filter((id) => id !== senseId)
        : [...addForm.selectedSenseIds, senseId],
    })
  }

  async function handleAddSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!addForm || addForm.selectedSenseIds.length === 0) return
    const data = await addEntry.execute(CREATE_ENTRY_FROM_CATALOG, {
      input: {
        refEntryId: addForm.refEntryId,
        senseIds: addForm.selectedSenseIds,
        notes: addForm.notes || null,
        createCard: addForm.createCard,
      },
    })
    if (data?.createEntryFromCatalog?.entry) {
      const created = data.createEntryFromCatalog.entry
      setAddSuccess(`Entry "${created.text}" added (id: ${created.id})`)
      setAddForm(null)
    }
  }

  // ---------- Render helpers ----------

  function renderSense(sense: Sense) {
    return (
      <div key={sense.id} className="ml-4 mb-2">
        <div className="text-sm">
          <span className="font-medium text-indigo-700">{sense.partOfSpeech}</span>
          {sense.cefrLevel && (
            <span className="ml-2 text-xs bg-yellow-100 text-yellow-800 px-1 rounded">
              {sense.cefrLevel}
            </span>
          )}
          <span className="ml-2 text-gray-700">{sense.definition}</span>
        </div>
        {sense.translations.length > 0 && (
          <div className="text-xs text-gray-500 ml-2">
            Translations: {sense.translations.map((t) => t.text).join(', ')}
          </div>
        )}
        {sense.examples.length > 0 && (
          <div className="text-xs text-gray-400 ml-2 italic">
            {sense.examples.map((ex) => (
              <div key={ex.id}>
                "{ex.sentence}"
                {ex.translation && <span className="not-italic"> -- {ex.translation}</span>}
              </div>
            ))}
          </div>
        )}
      </div>
    )
  }

  function renderResultCard(entry: RefEntry) {
    return (
      <div key={entry.id} className="bg-white border border-gray-200 rounded-lg p-4 shadow-sm">
        <div className="flex items-center justify-between mb-2">
          <h3 className="text-lg font-bold text-gray-900">{entry.text}</h3>
          <div className="space-x-2">
            <button
              onClick={() => handlePreview(entry.text)}
              disabled={preview.loading}
              className="text-xs px-3 py-1 bg-blue-50 text-blue-700 border border-blue-200 rounded hover:bg-blue-100 disabled:opacity-50"
            >
              {preview.loading ? 'Loading...' : 'Preview'}
            </button>
            <button
              onClick={() => handleStartAdd(entry)}
              className="text-xs px-3 py-1 bg-green-50 text-green-700 border border-green-200 rounded hover:bg-green-100"
            >
              Add to Dictionary
            </button>
          </div>
        </div>
        {entry.pronunciations.length > 0 && (
          <div className="text-xs text-gray-500 mb-2">
            {entry.pronunciations.map((p) => (
              <span key={p.id} className="mr-3">
                {p.region && <span className="text-gray-400">[{p.region}]</span>} {p.transcription}
              </span>
            ))}
          </div>
        )}
        {entry.senses.map(renderSense)}
      </div>
    )
  }

  // ---------- Main render ----------

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold text-gray-800">Reference Catalog</h1>
      <p className="text-sm text-gray-500">
        Search the reference catalog (public, no auth required). Add entries to your dictionary
        (requires auth).
      </p>

      {/* Search bar */}
      <form onSubmit={handleSearch} className="flex items-center gap-3">
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          placeholder="Search word or phrase..."
          className="flex-1 border border-gray-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
        />
        <select
          value={limit}
          onChange={(e) => setLimit(Number(e.target.value))}
          className="border border-gray-300 rounded px-2 py-2 text-sm bg-white"
        >
          <option value={5}>5</option>
          <option value={10}>10</option>
          <option value={20}>20</option>
        </select>
        <button
          type="submit"
          disabled={search.loading || !searchQuery.trim()}
          className="bg-blue-600 text-white px-4 py-2 rounded text-sm hover:bg-blue-700 disabled:opacity-50"
        >
          {search.loading ? 'Searching...' : 'Search'}
        </button>
      </form>

      {/* Errors */}
      {search.errors && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
          {search.errors.map((err, i) => (
            <div key={i}>{err.message}</div>
          ))}
        </div>
      )}

      {/* Success message */}
      {addSuccess && (
        <div className="bg-green-50 border border-green-200 text-green-700 text-sm rounded p-3">
          {addSuccess}
        </div>
      )}

      {/* Add entry errors */}
      {addEntry.errors && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
          <strong>Add entry error:</strong>
          {addEntry.errors.map((err, i) => (
            <div key={i}>{err.message}</div>
          ))}
        </div>
      )}

      {/* Search results */}
      {search.data?.searchCatalog && (
        <div className="space-y-4">
          <h2 className="text-sm font-medium text-gray-500">
            {search.data.searchCatalog.length} result(s)
          </h2>
          {search.data.searchCatalog.map(renderResultCard)}
        </div>
      )}

      {/* Preview panel */}
      {previewEntry && (
        <div className="border-2 border-blue-300 rounded-lg p-5 bg-blue-50">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-xl font-bold text-blue-900">Preview: {previewEntry.text}</h2>
            <button
              onClick={() => setPreviewEntry(null)}
              className="text-xs text-gray-500 hover:text-gray-700"
            >
              Close
            </button>
          </div>
          {previewEntry.pronunciations.length > 0 && (
            <div className="text-sm text-gray-600 mb-3">
              {previewEntry.pronunciations.map((p) => (
                <span key={p.id} className="mr-4">
                  {p.region && <span className="text-gray-400">[{p.region}]</span>} {p.transcription}
                  {p.audioUrl && (
                    <a
                      href={p.audioUrl}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="ml-1 text-blue-500 underline text-xs"
                    >
                      audio
                    </a>
                  )}
                </span>
              ))}
            </div>
          )}
          {previewEntry.images.length > 0 && (
            <div className="flex gap-2 mb-3">
              {previewEntry.images.map((img) => (
                <div key={img.id} className="text-center">
                  <img
                    src={img.url}
                    alt={img.caption}
                    className="h-24 rounded border"
                  />
                  {img.caption && (
                    <div className="text-xs text-gray-500">{img.caption}</div>
                  )}
                </div>
              ))}
            </div>
          )}
          <div className="space-y-1">
            <h3 className="text-sm font-semibold text-gray-700">Senses:</h3>
            {previewEntry.senses.map(renderSense)}
          </div>
          <div className="mt-3">
            <button
              onClick={() => handleStartAdd(previewEntry)}
              className="text-sm px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700"
            >
              Add to Dictionary
            </button>
          </div>
        </div>
      )}

      {/* Preview errors */}
      {preview.errors && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
          <strong>Preview error:</strong>
          {preview.errors.map((err, i) => (
            <div key={i}>{err.message}</div>
          ))}
        </div>
      )}

      {/* Add to Dictionary form */}
      {addForm && (
        <div className="border-2 border-green-300 rounded-lg p-5 bg-green-50">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-lg font-bold text-green-900">
              Add to Dictionary: {addForm.refEntryText}
            </h2>
            <button
              onClick={() => setAddForm(null)}
              className="text-xs text-gray-500 hover:text-gray-700"
            >
              Cancel
            </button>
          </div>

          {!isAuthenticated && (
            <div className="bg-yellow-50 border border-yellow-200 text-yellow-800 text-sm rounded p-3 mb-3">
              You must be logged in to add entries to your dictionary.
            </div>
          )}

          <form onSubmit={handleAddSubmit} className="space-y-4">
            {/* Sense checkboxes */}
            <div>
              <h3 className="text-sm font-semibold text-gray-700 mb-2">Select senses to include:</h3>
              {addForm.senses.map((sense) => (
                <label
                  key={sense.id}
                  className="flex items-start gap-2 mb-2 cursor-pointer"
                >
                  <input
                    type="checkbox"
                    checked={addForm.selectedSenseIds.includes(sense.id)}
                    onChange={() => toggleSense(sense.id)}
                    className="mt-1"
                  />
                  <div className="text-sm">
                    <span className="font-medium text-indigo-700">{sense.partOfSpeech}</span>
                    {sense.cefrLevel && (
                      <span className="ml-1 text-xs bg-yellow-100 text-yellow-800 px-1 rounded">
                        {sense.cefrLevel}
                      </span>
                    )}
                    <span className="ml-1 text-gray-700">{sense.definition}</span>
                    {sense.translations.length > 0 && (
                      <span className="ml-1 text-gray-500 text-xs">
                        ({sense.translations.map((t) => t.text).join(', ')})
                      </span>
                    )}
                  </div>
                </label>
              ))}
            </div>

            {/* Notes */}
            <div>
              <label className="block text-sm font-semibold text-gray-700 mb-1">
                Notes (optional)
              </label>
              <textarea
                value={addForm.notes}
                onChange={(e) => setAddForm({ ...addForm, notes: e.target.value })}
                rows={3}
                className="w-full border border-gray-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-green-400"
                placeholder="Add personal notes..."
              />
            </div>

            {/* Create card checkbox */}
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={addForm.createCard}
                onChange={(e) => setAddForm({ ...addForm, createCard: e.target.checked })}
              />
              <span className="text-sm text-gray-700">Create study card automatically</span>
            </label>

            {/* Submit */}
            <button
              type="submit"
              disabled={addEntry.loading || addForm.selectedSenseIds.length === 0}
              className="bg-green-600 text-white px-4 py-2 rounded text-sm hover:bg-green-700 disabled:opacity-50"
            >
              {addEntry.loading ? 'Adding...' : 'Add Entry'}
            </button>
          </form>
        </div>
      )}

      {/* Raw panel */}
      <RawPanel raw={lastRaw} />
    </div>
  )
}
