import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useGraphQL } from '../hooks/useGraphQL'
import { useAuth } from '../auth/useAuth'
import { RawPanel } from '../components/RawPanel'

// ---------- GraphQL queries / mutations ----------

const DICTIONARY_QUERY = `
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
}`

const DELETED_ENTRIES_QUERY = `
query DeletedEntries($limit: Int, $offset: Int) {
  deletedEntries(limit: $limit, offset: $offset) {
    entries { id text notes deletedAt }
    totalCount
  }
}`

const EXPORT_ENTRIES_QUERY = `
query ExportEntries {
  exportEntries {
    exportedAt
    items {
      text notes cardStatus createdAt
      senses {
        definition partOfSpeech
        translations
        examples { sentence translation }
      }
    }
  }
}`

const CREATE_CUSTOM_MUTATION = `
mutation CreateCustom($input: CreateEntryCustomInput!) {
  createEntryCustom(input: $input) {
    entry { id text notes }
  }
}`

const RESTORE_ENTRY_MUTATION = `
mutation RestoreEntry($id: UUID!) {
  restoreEntry(id: $id) { entry { id text } }
}`

const BATCH_DELETE_MUTATION = `
mutation BatchDelete($ids: [UUID!]!) {
  batchDeleteEntries(ids: $ids) { deletedCount errors { id message } }
}`

const IMPORT_MUTATION = `
mutation Import($input: ImportEntriesInput!) {
  importEntries(input: $input) { importedCount skippedCount errors { index text message } }
}`

// ---------- Types ----------

interface DictionaryEntry {
  id: string
  text: string
  textNormalized: string
  notes: string | null
  createdAt: string
  updatedAt: string
  deletedAt: string | null
  senses: { id: string; definition: string; partOfSpeech: string }[]
  card: { id: string; status: string; nextReviewAt: string } | null
  topics: { id: string; name: string }[]
}

interface DictionaryEdge {
  cursor: string
  node: DictionaryEntry
}

interface DictionaryData {
  dictionary: {
    edges: DictionaryEdge[]
    pageInfo: {
      hasNextPage: boolean
      hasPreviousPage: boolean
      startCursor: string | null
      endCursor: string | null
    }
    totalCount: number
  }
}

interface DeletedEntry {
  id: string
  text: string
  notes: string | null
  deletedAt: string
}

interface DeletedEntriesData {
  deletedEntries: {
    entries: DeletedEntry[]
    totalCount: number
  }
}

interface ExportData {
  exportEntries: {
    exportedAt: string
    items: unknown[]
  }
}

interface CreateCustomData {
  createEntryCustom: {
    entry: { id: string; text: string; notes: string | null }
  }
}

interface RestoreEntryData {
  restoreEntry: { entry: { id: string; text: string } }
}

interface BatchDeleteData {
  batchDeleteEntries: {
    deletedCount: number
    errors: { id: string; message: string }[]
  }
}

interface ImportData {
  importEntries: {
    importedCount: number
    skippedCount: number
    errors: { index: number; text: string; message: string }[]
  }
}

interface CustomSense {
  definition: string
  partOfSpeech: string
  translations: string[]
  examples: { sentence: string; translation: string }[]
}

interface CreateCustomForm {
  text: string
  senses: CustomSense[]
  notes: string
  createCard: boolean
  topicId: string
}

// ---------- Constants ----------

const PART_OF_SPEECH_OPTIONS = [
  'All', 'NOUN', 'VERB', 'ADJECTIVE', 'ADVERB', 'PRONOUN',
  'PREPOSITION', 'CONJUNCTION', 'INTERJECTION', 'PHRASE', 'IDIOM', 'OTHER',
]

const STATUS_OPTIONS = ['All', 'NEW', 'LEARNING', 'REVIEW', 'MASTERED']

const SORT_FIELD_OPTIONS = ['TEXT', 'CREATED_AT', 'UPDATED_AT']

// ---------- Helper ----------

function emptySense(): CustomSense {
  return { definition: '', partOfSpeech: 'NOUN', translations: [''], examples: [] }
}

function emptyCreateForm(): CreateCustomForm {
  return { text: '', senses: [emptySense()], notes: '', createCard: false, topicId: '' }
}

// ---------- Component ----------

export function DictionaryPage() {
  const navigate = useNavigate()
  const { isAuthenticated } = useAuth()

  // Tab state
  const [activeTab, setActiveTab] = useState<'active' | 'deleted'>('active')

  // --- Active entries state ---
  // Filters
  const [searchText, setSearchText] = useState('')
  const [hasCard, setHasCard] = useState<'all' | 'with' | 'without'>('all')
  const [partOfSpeech, setPartOfSpeech] = useState('All')
  const [status, setStatus] = useState('All')
  const [topicId, setTopicId] = useState('')
  const [sortField, setSortField] = useState('CREATED_AT')
  const [sortDirection, setSortDirection] = useState<'ASC' | 'DESC'>('DESC')

  // Pagination
  const [paginationMode, setPaginationMode] = useState<'cursor' | 'offset'>('cursor')
  const [cursorFirst, setCursorFirst] = useState(20)
  const [cursorAfter, setCursorAfter] = useState('')
  const [offsetLimit, setOffsetLimit] = useState(20)
  const [offsetOffset, setOffsetOffset] = useState(0)

  // Selection
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())

  // Forms visibility
  const [showCreateForm, setShowCreateForm] = useState(false)
  const [showImportForm, setShowImportForm] = useState(false)
  const [createForm, setCreateForm] = useState<CreateCustomForm>(emptyCreateForm())
  const [importJson, setImportJson] = useState('')

  // Messages
  const [successMsg, setSuccessMsg] = useState<string | null>(null)

  // GraphQL hooks
  const dictionary = useGraphQL<DictionaryData>()
  const deletedEntries = useGraphQL<DeletedEntriesData>()
  const createCustom = useGraphQL<CreateCustomData>()
  const restoreEntry = useGraphQL<RestoreEntryData>()
  const batchDelete = useGraphQL<BatchDeleteData>()
  const importEntries = useGraphQL<ImportData>()
  const exportEntries = useGraphQL<ExportData>()

  // --- Deleted entries state ---
  const [deletedLimit, setDeletedLimit] = useState(20)
  const [deletedOffset, setDeletedOffset] = useState(0)

  // The raw panel shows the most recent operation
  const lastRaw = createCustom.raw ?? restoreEntry.raw
    ?? batchDelete.raw ?? importEntries.raw ?? exportEntries.raw
    ?? deletedEntries.raw ?? dictionary.raw

  // Auto-load dictionary on mount
  useEffect(() => {
    if (isAuthenticated) {
      dictionary.execute(DICTIONARY_QUERY, { input: buildFilterInput() })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // ---------- Handlers ----------

  function buildFilterInput() {
    const input: Record<string, unknown> = {}

    if (searchText.trim()) input.searchText = searchText.trim()
    if (hasCard === 'with') input.hasCard = true
    if (hasCard === 'without') input.hasCard = false
    if (partOfSpeech !== 'All') input.partOfSpeech = partOfSpeech
    if (status !== 'All') input.status = status
    if (topicId.trim()) input.topicId = topicId.trim()

    input.sortField = sortField
    input.sortDirection = sortDirection

    if (paginationMode === 'cursor') {
      input.first = cursorFirst
      if (cursorAfter.trim()) input.after = cursorAfter.trim()
    } else {
      input.limit = offsetLimit
      input.offset = offsetOffset
    }

    return input
  }

  async function handleApplyFilters() {
    setSuccessMsg(null)
    setSelectedIds(new Set())
    await dictionary.execute(DICTIONARY_QUERY, { input: buildFilterInput() })
  }

  async function handleNextPageCursor() {
    const endCursor = dictionary.data?.dictionary.pageInfo.endCursor
    if (endCursor) {
      setCursorAfter(endCursor)
      setSelectedIds(new Set())
      await dictionary.execute(DICTIONARY_QUERY, {
        input: { ...buildFilterInput(), after: endCursor, first: cursorFirst },
      })
    }
  }

  async function handleOffsetNext() {
    const newOffset = offsetOffset + offsetLimit
    setOffsetOffset(newOffset)
    setSelectedIds(new Set())
    await dictionary.execute(DICTIONARY_QUERY, {
      input: { ...buildFilterInput(), offset: newOffset, limit: offsetLimit },
    })
  }

  async function handleOffsetPrev() {
    const newOffset = Math.max(0, offsetOffset - offsetLimit)
    setOffsetOffset(newOffset)
    setSelectedIds(new Set())
    await dictionary.execute(DICTIONARY_QUERY, {
      input: { ...buildFilterInput(), offset: newOffset, limit: offsetLimit },
    })
  }

  function toggleSelection(id: string) {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  function toggleSelectAll() {
    const edges = dictionary.data?.dictionary.edges ?? []
    if (selectedIds.size === edges.length) {
      setSelectedIds(new Set())
    } else {
      setSelectedIds(new Set(edges.map((e) => e.node.id)))
    }
  }

  async function handleCreateCustom(e: React.FormEvent) {
    e.preventDefault()
    setSuccessMsg(null)

    const input: Record<string, unknown> = {
      text: createForm.text,
      senses: createForm.senses.map((s) => ({
        definition: s.definition || undefined,
        partOfSpeech: s.partOfSpeech || undefined,
        translations: s.translations.filter((t) => t.trim()),
        examples: s.examples.filter((ex) => ex.sentence.trim()).map((ex) => ({
          sentence: ex.sentence,
          translation: ex.translation || undefined,
        })),
      })),
    }
    if (createForm.notes.trim()) input.notes = createForm.notes.trim()
    if (createForm.createCard) input.createCard = true
    if (createForm.topicId.trim()) input.topicId = createForm.topicId.trim()

    const data = await createCustom.execute(CREATE_CUSTOM_MUTATION, { input })
    if (data?.createEntryCustom?.entry) {
      setSuccessMsg(`Entry "${data.createEntryCustom.entry.text}" created (id: ${data.createEntryCustom.entry.id})`)
      setCreateForm(emptyCreateForm())
      setShowCreateForm(false)
    }
  }

  async function handleImport(e: React.FormEvent) {
    e.preventDefault()
    setSuccessMsg(null)
    try {
      const items = JSON.parse(importJson)
      if (!Array.isArray(items)) {
        setSuccessMsg('Error: JSON must be an array')
        return
      }
      const data = await importEntries.execute(IMPORT_MUTATION, { input: { items } })
      if (data?.importEntries) {
        const r = data.importEntries
        setSuccessMsg(`Import done: ${r.importedCount} imported, ${r.skippedCount} skipped, ${r.errors.length} errors`)
        if (r.importedCount > 0) setImportJson('')
      }
    } catch {
      setSuccessMsg('Error: invalid JSON')
    }
  }

  async function handleExport() {
    setSuccessMsg(null)
    const data = await exportEntries.execute(EXPORT_ENTRIES_QUERY)
    if (data?.exportEntries) {
      const blob = new Blob([JSON.stringify(data.exportEntries, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `dictionary-export-${data.exportEntries.exportedAt || 'now'}.json`
      a.click()
      URL.revokeObjectURL(url)
      setSuccessMsg('Export downloaded')
    }
  }

  async function handleBatchDelete() {
    if (selectedIds.size === 0) return
    setSuccessMsg(null)
    const ids = Array.from(selectedIds)
    const data = await batchDelete.execute(BATCH_DELETE_MUTATION, { ids })
    if (data?.batchDeleteEntries) {
      const r = data.batchDeleteEntries
      setSuccessMsg(`Deleted ${r.deletedCount} entries` + (r.errors.length > 0 ? `, ${r.errors.length} errors` : ''))
      setSelectedIds(new Set())
      await handleApplyFilters()
    }
  }

  // --- Deleted tab handlers ---

  async function handleFetchDeleted() {
    setSuccessMsg(null)
    await deletedEntries.execute(DELETED_ENTRIES_QUERY, { limit: deletedLimit, offset: deletedOffset })
  }

  async function handleRestore(id: string) {
    setSuccessMsg(null)
    const data = await restoreEntry.execute(RESTORE_ENTRY_MUTATION, { id })
    if (data?.restoreEntry?.entry) {
      setSuccessMsg(`Restored "${data.restoreEntry.entry.text}"`)
      await handleFetchDeleted()
    }
  }

  // ---------- Sense form helpers ----------

  function updateSense(index: number, field: keyof CustomSense, value: unknown) {
    setCreateForm((prev) => {
      const senses = [...prev.senses]
      senses[index] = { ...senses[index], [field]: value }
      return { ...prev, senses }
    })
  }

  function addSense() {
    setCreateForm((prev) => ({ ...prev, senses: [...prev.senses, emptySense()] }))
  }

  function removeSense(index: number) {
    setCreateForm((prev) => ({
      ...prev,
      senses: prev.senses.filter((_, i) => i !== index),
    }))
  }

  function updateTranslation(senseIdx: number, transIdx: number, value: string) {
    setCreateForm((prev) => {
      const senses = [...prev.senses]
      const translations = [...senses[senseIdx].translations]
      translations[transIdx] = value
      senses[senseIdx] = { ...senses[senseIdx], translations }
      return { ...prev, senses }
    })
  }

  function addTranslation(senseIdx: number) {
    setCreateForm((prev) => {
      const senses = [...prev.senses]
      senses[senseIdx] = { ...senses[senseIdx], translations: [...senses[senseIdx].translations, ''] }
      return { ...prev, senses }
    })
  }

  function removeTranslation(senseIdx: number, transIdx: number) {
    setCreateForm((prev) => {
      const senses = [...prev.senses]
      senses[senseIdx] = {
        ...senses[senseIdx],
        translations: senses[senseIdx].translations.filter((_, i) => i !== transIdx),
      }
      return { ...prev, senses }
    })
  }

  // ---------- Render: Active Entries Tab ----------

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

  function renderFilterBar() {
    return (
      <div className="bg-white border border-gray-200 rounded-lg p-4 space-y-3">
        <h3 className="text-sm font-semibold text-gray-700">Filters</h3>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          <div>
            <label className="block text-xs text-gray-500 mb-1">Search text</label>
            <input
              type="text"
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              placeholder="word or phrase..."
              className="w-full border border-gray-300 rounded px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">Has Card</label>
            <select
              value={hasCard}
              onChange={(e) => setHasCard(e.target.value as 'all' | 'with' | 'without')}
              className="w-full border border-gray-300 rounded px-2 py-1.5 text-sm bg-white"
            >
              <option value="all">All</option>
              <option value="with">With card</option>
              <option value="without">Without card</option>
            </select>
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">Part of Speech</label>
            <select
              value={partOfSpeech}
              onChange={(e) => setPartOfSpeech(e.target.value)}
              className="w-full border border-gray-300 rounded px-2 py-1.5 text-sm bg-white"
            >
              {PART_OF_SPEECH_OPTIONS.map((opt) => (
                <option key={opt} value={opt}>{opt}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">Status</label>
            <select
              value={status}
              onChange={(e) => setStatus(e.target.value)}
              className="w-full border border-gray-300 rounded px-2 py-1.5 text-sm bg-white"
            >
              {STATUS_OPTIONS.map((opt) => (
                <option key={opt} value={opt}>{opt}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">Topic ID (UUID)</label>
            <input
              type="text"
              value={topicId}
              onChange={(e) => setTopicId(e.target.value)}
              placeholder="uuid..."
              className="w-full border border-gray-300 rounded px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">Sort Field</label>
            <select
              value={sortField}
              onChange={(e) => setSortField(e.target.value)}
              className="w-full border border-gray-300 rounded px-2 py-1.5 text-sm bg-white"
            >
              {SORT_FIELD_OPTIONS.map((opt) => (
                <option key={opt} value={opt}>{opt}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">Sort Direction</label>
            <button
              type="button"
              onClick={() => setSortDirection((d) => d === 'ASC' ? 'DESC' : 'ASC')}
              className="w-full border border-gray-300 rounded px-2 py-1.5 text-sm bg-white hover:bg-gray-50 text-left"
            >
              {sortDirection}
            </button>
          </div>
          <div className="flex items-end">
            <button
              onClick={handleApplyFilters}
              disabled={dictionary.loading}
              className="w-full bg-blue-600 text-white px-4 py-1.5 rounded text-sm hover:bg-blue-700 disabled:opacity-50"
            >
              {dictionary.loading ? 'Loading...' : 'Apply Filters'}
            </button>
          </div>
        </div>
      </div>
    )
  }

  function renderPagination() {
    return (
      <div className="bg-white border border-gray-200 rounded-lg p-4 space-y-3">
        <div className="flex items-center gap-4">
          <h3 className="text-sm font-semibold text-gray-700">Pagination</h3>
          <div className="flex items-center gap-2">
            <label className="text-xs text-gray-500">
              <input
                type="radio"
                name="paginationMode"
                checked={paginationMode === 'cursor'}
                onChange={() => setPaginationMode('cursor')}
                className="mr-1"
              />
              Cursor
            </label>
            <label className="text-xs text-gray-500">
              <input
                type="radio"
                name="paginationMode"
                checked={paginationMode === 'offset'}
                onChange={() => setPaginationMode('offset')}
                className="mr-1"
              />
              Offset
            </label>
          </div>
          {dictionary.data && (
            <span className="text-xs text-gray-500 ml-auto">
              Total: {dictionary.data.dictionary.totalCount}
            </span>
          )}
        </div>

        {paginationMode === 'cursor' ? (
          <div className="flex items-center gap-3">
            <div>
              <label className="block text-xs text-gray-500 mb-1">first</label>
              <input
                type="number"
                min={1}
                max={100}
                value={cursorFirst}
                onChange={(e) => setCursorFirst(Number(e.target.value))}
                className="w-20 border border-gray-300 rounded px-2 py-1 text-sm"
              />
            </div>
            <div className="flex-1">
              <label className="block text-xs text-gray-500 mb-1">after (endCursor)</label>
              <input
                type="text"
                value={cursorAfter}
                onChange={(e) => setCursorAfter(e.target.value)}
                placeholder={dictionary.data?.dictionary.pageInfo.endCursor ?? 'cursor...'}
                className="w-full border border-gray-300 rounded px-2 py-1 text-sm"
              />
            </div>
            <div className="flex items-end">
              <button
                onClick={handleNextPageCursor}
                disabled={dictionary.loading || !dictionary.data?.dictionary.pageInfo.hasNextPage}
                className="px-3 py-1 text-sm border border-gray-300 rounded hover:bg-gray-50 disabled:opacity-50"
              >
                Next Page
              </button>
            </div>
          </div>
        ) : (
          <div className="flex items-center gap-3">
            <div>
              <label className="block text-xs text-gray-500 mb-1">limit</label>
              <input
                type="number"
                min={1}
                max={100}
                value={offsetLimit}
                onChange={(e) => setOffsetLimit(Number(e.target.value))}
                className="w-20 border border-gray-300 rounded px-2 py-1 text-sm"
              />
            </div>
            <div>
              <label className="block text-xs text-gray-500 mb-1">offset</label>
              <input
                type="number"
                min={0}
                value={offsetOffset}
                onChange={(e) => setOffsetOffset(Number(e.target.value))}
                className="w-20 border border-gray-300 rounded px-2 py-1 text-sm"
              />
            </div>
            <div className="flex items-end gap-2">
              <button
                onClick={handleOffsetPrev}
                disabled={dictionary.loading || offsetOffset === 0}
                className="px-3 py-1 text-sm border border-gray-300 rounded hover:bg-gray-50 disabled:opacity-50"
              >
                Prev
              </button>
              <button
                onClick={handleOffsetNext}
                disabled={dictionary.loading || !dictionary.data?.dictionary.pageInfo.hasNextPage}
                className="px-3 py-1 text-sm border border-gray-300 rounded hover:bg-gray-50 disabled:opacity-50"
              >
                Next
              </button>
            </div>
          </div>
        )}
      </div>
    )
  }

  function renderEntryTable() {
    const edges = dictionary.data?.dictionary.edges ?? []
    if (edges.length === 0 && dictionary.data) {
      return (
        <div className="text-center py-8">
          <div className="text-gray-400 text-sm">No entries found.</div>
          <div className="text-gray-400 text-xs mt-1">
            Visit the Catalog to search and add your first words, or adjust your filters.
          </div>
        </div>
      )
    }
    if (!dictionary.data) return null

    return (
      <div className="overflow-x-auto">
        <table className="w-full text-sm border border-gray-200 rounded-lg">
          <thead className="bg-gray-100">
            <tr>
              <th className="px-3 py-2 text-left w-8">
                <input
                  type="checkbox"
                  checked={selectedIds.size > 0 && selectedIds.size === edges.length}
                  onChange={toggleSelectAll}
                />
              </th>
              <th className="px-3 py-2 text-left">Text</th>
              <th className="px-3 py-2 text-left">Notes</th>
              <th className="px-3 py-2 text-left">Card Status</th>
              <th className="px-3 py-2 text-left">Senses</th>
              <th className="px-3 py-2 text-left">Created</th>
            </tr>
          </thead>
          <tbody>
            {edges.map((edge) => {
              const entry = edge.node
              const truncatedNotes = entry.notes
                ? (entry.notes.length > 50 ? entry.notes.slice(0, 50) + '...' : entry.notes)
                : '-'
              const cardStatus = entry.card?.status ?? 'NO CARD'
              const statusColors: Record<string, string> = {
                NEW: 'bg-blue-100 text-blue-800',
                LEARNING: 'bg-yellow-100 text-yellow-800',
                REVIEW: 'bg-purple-100 text-purple-800',
                MASTERED: 'bg-green-100 text-green-800',
                'NO CARD': 'bg-gray-100 text-gray-600',
              }
              return (
                <tr
                  key={entry.id}
                  className="border-t border-gray-100 hover:bg-gray-50 cursor-pointer"
                  onClick={() => navigate(`/entry/${entry.id}`)}
                >
                  <td className="px-3 py-2" onClick={(e) => e.stopPropagation()}>
                    <input
                      type="checkbox"
                      checked={selectedIds.has(entry.id)}
                      onChange={() => toggleSelection(entry.id)}
                    />
                  </td>
                  <td className="px-3 py-2 font-medium text-gray-900">{entry.text}</td>
                  <td className="px-3 py-2 text-gray-500">{truncatedNotes}</td>
                  <td className="px-3 py-2">
                    <span className={`text-xs px-2 py-0.5 rounded ${statusColors[cardStatus] ?? 'bg-gray-100 text-gray-600'}`}>
                      {cardStatus}
                    </span>
                  </td>
                  <td className="px-3 py-2 text-gray-500">{entry.senses.length}</td>
                  <td className="px-3 py-2 text-gray-500 text-xs">
                    {new Date(entry.createdAt).toLocaleDateString()}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    )
  }

  function renderActionsBar() {
    return (
      <div className="flex flex-wrap gap-2">
        <button
          onClick={() => { setShowCreateForm(!showCreateForm); setShowImportForm(false) }}
          className="px-3 py-1.5 text-sm bg-green-600 text-white rounded hover:bg-green-700"
        >
          {showCreateForm ? 'Hide Create Form' : 'Create Custom Entry'}
        </button>
        <button
          onClick={() => { setShowImportForm(!showImportForm); setShowCreateForm(false) }}
          className="px-3 py-1.5 text-sm bg-indigo-600 text-white rounded hover:bg-indigo-700"
        >
          {showImportForm ? 'Hide Import Form' : 'Import Entries'}
        </button>
        <button
          onClick={handleExport}
          disabled={exportEntries.loading}
          className="px-3 py-1.5 text-sm bg-gray-600 text-white rounded hover:bg-gray-700 disabled:opacity-50"
        >
          {exportEntries.loading ? 'Exporting...' : 'Export All'}
        </button>
        {selectedIds.size > 0 && (
          <button
            onClick={handleBatchDelete}
            disabled={batchDelete.loading}
            className="px-3 py-1.5 text-sm bg-red-600 text-white rounded hover:bg-red-700 disabled:opacity-50"
          >
            {batchDelete.loading ? 'Deleting...' : `Delete Selected (${selectedIds.size})`}
          </button>
        )}
      </div>
    )
  }

  function renderCreateForm() {
    if (!showCreateForm) return null

    return (
      <div className="border-2 border-green-300 rounded-lg p-5 bg-green-50 space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-bold text-green-900">Create Custom Entry</h3>
          <button onClick={() => setShowCreateForm(false)} className="text-xs text-gray-500 hover:text-gray-700">
            Close
          </button>
        </div>

        {!isAuthenticated && (
          <div className="bg-yellow-50 border border-yellow-200 text-yellow-800 text-sm rounded p-3">
            You must be logged in to create entries.
          </div>
        )}

        <form onSubmit={handleCreateCustom} className="space-y-4">
          <div>
            <label className="block text-sm font-semibold text-gray-700 mb-1">Text *</label>
            <input
              type="text"
              value={createForm.text}
              onChange={(e) => setCreateForm({ ...createForm, text: e.target.value })}
              required
              className="w-full border border-gray-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-green-400"
              placeholder="Enter word or phrase..."
            />
          </div>

          {/* Senses */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <label className="text-sm font-semibold text-gray-700">Senses *</label>
              <button type="button" onClick={addSense} className="text-xs px-2 py-1 bg-green-200 text-green-800 rounded hover:bg-green-300">
                + Add Sense
              </button>
            </div>
            {createForm.senses.map((sense, senseIdx) => (
              <div key={senseIdx} className="border border-green-200 rounded p-3 bg-white space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-medium text-gray-500">Sense #{senseIdx + 1}</span>
                  {createForm.senses.length > 1 && (
                    <button type="button" onClick={() => removeSense(senseIdx)} className="text-xs text-red-500 hover:text-red-700">
                      Remove
                    </button>
                  )}
                </div>
                <div className="grid grid-cols-2 gap-2">
                  <div>
                    <label className="block text-xs text-gray-500 mb-0.5">Definition</label>
                    <input
                      type="text"
                      value={sense.definition}
                      onChange={(e) => updateSense(senseIdx, 'definition', e.target.value)}
                      className="w-full border border-gray-300 rounded px-2 py-1 text-sm"
                      placeholder="definition..."
                    />
                  </div>
                  <div>
                    <label className="block text-xs text-gray-500 mb-0.5">Part of Speech</label>
                    <select
                      value={sense.partOfSpeech}
                      onChange={(e) => updateSense(senseIdx, 'partOfSpeech', e.target.value)}
                      className="w-full border border-gray-300 rounded px-2 py-1 text-sm bg-white"
                    >
                      {PART_OF_SPEECH_OPTIONS.filter((o) => o !== 'All').map((opt) => (
                        <option key={opt} value={opt}>{opt}</option>
                      ))}
                    </select>
                  </div>
                </div>

                {/* Translations */}
                <div>
                  <div className="flex items-center justify-between mb-1">
                    <label className="text-xs text-gray-500">Translations</label>
                    <button type="button" onClick={() => addTranslation(senseIdx)} className="text-xs text-blue-600 hover:text-blue-800">
                      + Add
                    </button>
                  </div>
                  {sense.translations.map((trans, transIdx) => (
                    <div key={transIdx} className="flex items-center gap-1 mb-1">
                      <input
                        type="text"
                        value={trans}
                        onChange={(e) => updateTranslation(senseIdx, transIdx, e.target.value)}
                        className="flex-1 border border-gray-300 rounded px-2 py-1 text-sm"
                        placeholder="translation..."
                      />
                      {sense.translations.length > 1 && (
                        <button type="button" onClick={() => removeTranslation(senseIdx, transIdx)} className="text-xs text-red-400 hover:text-red-600 px-1">
                          x
                        </button>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>

          {/* Notes */}
          <div>
            <label className="block text-sm font-semibold text-gray-700 mb-1">Notes</label>
            <textarea
              value={createForm.notes}
              onChange={(e) => setCreateForm({ ...createForm, notes: e.target.value })}
              rows={2}
              className="w-full border border-gray-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-green-400"
              placeholder="Personal notes..."
            />
          </div>

          {/* Create card + Topic */}
          <div className="flex items-center gap-4">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={createForm.createCard}
                onChange={(e) => setCreateForm({ ...createForm, createCard: e.target.checked })}
              />
              <span className="text-sm text-gray-700">Create study card</span>
            </label>
            <div className="flex-1">
              <input
                type="text"
                value={createForm.topicId}
                onChange={(e) => setCreateForm({ ...createForm, topicId: e.target.value })}
                className="w-full border border-gray-300 rounded px-2 py-1 text-sm"
                placeholder="Topic ID (UUID, optional)"
              />
            </div>
          </div>

          <button
            type="submit"
            disabled={createCustom.loading || !createForm.text.trim() || createForm.senses.length === 0}
            className="bg-green-600 text-white px-4 py-2 rounded text-sm hover:bg-green-700 disabled:opacity-50"
          >
            {createCustom.loading ? 'Creating...' : 'Create Entry'}
          </button>
        </form>
      </div>
    )
  }

  function renderImportForm() {
    if (!showImportForm) return null

    return (
      <div className="border-2 border-indigo-300 rounded-lg p-5 bg-indigo-50 space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-bold text-indigo-900">Import Entries</h3>
          <button onClick={() => setShowImportForm(false)} className="text-xs text-gray-500 hover:text-gray-700">
            Close
          </button>
        </div>

        {!isAuthenticated && (
          <div className="bg-yellow-50 border border-yellow-200 text-yellow-800 text-sm rounded p-3">
            You must be logged in to import entries.
          </div>
        )}

        <form onSubmit={handleImport} className="space-y-3">
          <div>
            <label className="block text-sm font-semibold text-gray-700 mb-1">
              JSON Array of items
            </label>
            <p className="text-xs text-gray-500 mb-2">
              Format: [&#123;"text": "word", "translations": ["trans1"], "notes": "...", "topicName": "..."&#125;, ...]
            </p>
            <textarea
              value={importJson}
              onChange={(e) => setImportJson(e.target.value)}
              rows={6}
              className="w-full border border-gray-300 rounded px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-400"
              placeholder='[{"text": "hello", "translations": ["привет"]}]'
            />
          </div>
          <button
            type="submit"
            disabled={importEntries.loading || !importJson.trim()}
            className="bg-indigo-600 text-white px-4 py-2 rounded text-sm hover:bg-indigo-700 disabled:opacity-50"
          >
            {importEntries.loading ? 'Importing...' : 'Import'}
          </button>
        </form>
      </div>
    )
  }

  // ---------- Render: Deleted Entries Tab ----------

  function renderDeletedTab() {
    return (
      <div className="space-y-4">
        <div className="flex items-center gap-3">
          <div>
            <label className="block text-xs text-gray-500 mb-1">limit</label>
            <input
              type="number"
              min={1}
              max={100}
              value={deletedLimit}
              onChange={(e) => setDeletedLimit(Number(e.target.value))}
              className="w-20 border border-gray-300 rounded px-2 py-1 text-sm"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">offset</label>
            <input
              type="number"
              min={0}
              value={deletedOffset}
              onChange={(e) => setDeletedOffset(Number(e.target.value))}
              className="w-20 border border-gray-300 rounded px-2 py-1 text-sm"
            />
          </div>
          <div className="flex items-end">
            <button
              onClick={handleFetchDeleted}
              disabled={deletedEntries.loading}
              className="px-4 py-1 text-sm bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
            >
              {deletedEntries.loading ? 'Loading...' : 'Fetch Deleted'}
            </button>
          </div>
          {deletedEntries.data && (
            <div className="flex items-end">
              <span className="text-xs text-gray-500">
                Total: {deletedEntries.data.deletedEntries.totalCount}
              </span>
            </div>
          )}
        </div>

        {deletedEntries.data && (
          <div className="space-y-2">
            {deletedEntries.data.deletedEntries.entries.length === 0 ? (
              <div className="text-gray-500 text-sm">No deleted entries.</div>
            ) : (
              deletedEntries.data.deletedEntries.entries.map((entry) => (
                <div key={entry.id} className="bg-white border border-gray-200 rounded-lg p-3 flex items-center justify-between">
                  <div>
                    <span className="font-medium text-gray-900">{entry.text}</span>
                    {entry.notes && (
                      <span className="ml-2 text-gray-500 text-sm">{entry.notes}</span>
                    )}
                    <span className="ml-2 text-xs text-red-500">
                      deleted: {new Date(entry.deletedAt).toLocaleDateString()}
                    </span>
                  </div>
                  <button
                    onClick={() => handleRestore(entry.id)}
                    disabled={restoreEntry.loading}
                    className="px-3 py-1 text-sm bg-green-600 text-white rounded hover:bg-green-700 disabled:opacity-50"
                  >
                    {restoreEntry.loading ? '...' : 'Restore'}
                  </button>
                </div>
              ))
            )}
          </div>
        )}

        {deletedEntries.data && deletedEntries.data.deletedEntries.totalCount > deletedLimit && (
          <div className="flex gap-2">
            <button
              onClick={() => { setDeletedOffset(Math.max(0, deletedOffset - deletedLimit)); handleFetchDeleted() }}
              disabled={deletedEntries.loading || deletedOffset === 0}
              className="px-3 py-1 text-sm border border-gray-300 rounded hover:bg-gray-50 disabled:opacity-50"
            >
              Prev
            </button>
            <button
              onClick={() => { setDeletedOffset(deletedOffset + deletedLimit); handleFetchDeleted() }}
              disabled={deletedEntries.loading || deletedOffset + deletedLimit >= deletedEntries.data.deletedEntries.totalCount}
              className="px-3 py-1 text-sm border border-gray-300 rounded hover:bg-gray-50 disabled:opacity-50"
            >
              Next
            </button>
          </div>
        )}
      </div>
    )
  }

  // ---------- Main render ----------

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold text-gray-800">Dictionary</h1>
      <p className="text-sm text-gray-500">
        Manage your vocabulary entries. Data loads automatically when authenticated.
      </p>

      {/* Tab switcher */}
      <div className="flex gap-1 border-b border-gray-200">
        <button
          onClick={() => setActiveTab('active')}
          className={`px-4 py-2 text-sm font-medium rounded-t-lg ${
            activeTab === 'active'
              ? 'bg-white border border-gray-200 border-b-white text-blue-700 -mb-px'
              : 'text-gray-500 hover:text-gray-700'
          }`}
        >
          Active Entries
        </button>
        <button
          onClick={() => setActiveTab('deleted')}
          className={`px-4 py-2 text-sm font-medium rounded-t-lg ${
            activeTab === 'deleted'
              ? 'bg-white border border-gray-200 border-b-white text-red-700 -mb-px'
              : 'text-gray-500 hover:text-gray-700'
          }`}
        >
          Deleted (Trash)
        </button>
      </div>

      {/* Success / error messages */}
      {successMsg && (
        <div className="bg-green-50 border border-green-200 text-green-700 text-sm rounded p-3">
          {successMsg}
        </div>
      )}

      {/* GraphQL errors */}
      {dictionary.errors && activeTab === 'active' && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
          {dictionary.errors.map((err, i) => <div key={i}>{err.message}</div>)}
        </div>
      )}
      {createCustom.errors && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
          <strong>Create error: </strong>
          {createCustom.errors.map((err, i) => <div key={i}>{err.message}</div>)}
        </div>
      )}
      {importEntries.errors && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
          <strong>Import error: </strong>
          {importEntries.errors.map((err, i) => <div key={i}>{err.message}</div>)}
        </div>
      )}
      {batchDelete.errors && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
          <strong>Batch delete error: </strong>
          {batchDelete.errors.map((err, i) => <div key={i}>{err.message}</div>)}
        </div>
      )}
      {exportEntries.errors && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
          <strong>Export error: </strong>
          {exportEntries.errors.map((err, i) => <div key={i}>{err.message}</div>)}
        </div>
      )}
      {deletedEntries.errors && activeTab === 'deleted' && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
          {deletedEntries.errors.map((err, i) => <div key={i}>{err.message}</div>)}
        </div>
      )}
      {restoreEntry.errors && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
          <strong>Restore error: </strong>
          {restoreEntry.errors.map((err, i) => <div key={i}>{err.message}</div>)}
        </div>
      )}

      {/* Active tab content */}
      {activeTab === 'active' && (
        <div className="space-y-4">
          {renderSummaryBar()}
          {renderFilterBar()}
          {renderPagination()}
          {renderActionsBar()}
          {renderCreateForm()}
          {renderImportForm()}
          {renderEntryTable()}
        </div>
      )}

      {/* Deleted tab content */}
      {activeTab === 'deleted' && renderDeletedTab()}

      {/* Raw panel */}
      <RawPanel raw={lastRaw} />
    </div>
  )
}
