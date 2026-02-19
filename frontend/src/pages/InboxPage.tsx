import { useState, useEffect } from 'react'
import { useGraphQL } from '../hooks/useGraphQL'
import { useAuth } from '../auth/useAuth'
import { RawPanel } from '../components/RawPanel'

// ---------- GraphQL queries / mutations ----------

const INBOX_ITEMS_QUERY = `
query InboxItems($limit: Int, $offset: Int) {
  inboxItems(limit: $limit, offset: $offset) {
    items { id text context createdAt }
    totalCount
  }
}`

const CREATE_INBOX_ITEM_MUTATION = `
mutation CreateInboxItem($input: CreateInboxItemInput!) {
  createInboxItem(input: $input) { item { id text context createdAt } }
}`

const DELETE_INBOX_ITEM_MUTATION = `
mutation DeleteInboxItem($id: UUID!) {
  deleteInboxItem(id: $id) { itemId }
}`

const CLEAR_INBOX_MUTATION = `
mutation ClearInbox {
  clearInbox { deletedCount }
}`

// ---------- Types ----------

interface InboxItem {
  id: string
  text: string
  context: string | null
  createdAt: string
}

interface InboxItemsData {
  inboxItems: {
    items: InboxItem[]
    totalCount: number
  }
}

interface CreateInboxItemData {
  createInboxItem: {
    item: InboxItem
  }
}

interface DeleteInboxItemData {
  deleteInboxItem: {
    itemId: string
  }
}

interface ClearInboxData {
  clearInbox: {
    deletedCount: number
  }
}

// ---------- Component ----------

export function InboxPage() {
  const { isAuthenticated } = useAuth()

  // Create form state
  const [text, setText] = useState('')
  const [context, setContext] = useState('')

  // Pagination state
  const [limit, setLimit] = useState(20)
  const [offset, setOffset] = useState(0)

  // Messages
  const [successMsg, setSuccessMsg] = useState<string | null>(null)

  // GraphQL hooks
  const listItems = useGraphQL<InboxItemsData>()
  const createItem = useGraphQL<CreateInboxItemData>()
  const deleteItem = useGraphQL<DeleteInboxItemData>()
  const clearInbox = useGraphQL<ClearInboxData>()

  // The raw panel shows the most recent operation
  const lastRaw = createItem.raw ?? deleteItem.raw ?? clearInbox.raw ?? listItems.raw

  // Auto-load inbox items on mount
  useEffect(() => {
    if (isAuthenticated) {
      listItems.execute(INBOX_ITEMS_QUERY, { limit, offset })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // ---------- Handlers ----------

  async function handleFetch() {
    setSuccessMsg(null)
    await listItems.execute(INBOX_ITEMS_QUERY, { limit, offset })
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    setSuccessMsg(null)

    const input: Record<string, unknown> = { text: text.trim() }
    if (context.trim()) input.context = context.trim()

    const data = await createItem.execute(CREATE_INBOX_ITEM_MUTATION, { input })
    if (data?.createInboxItem?.item) {
      setSuccessMsg(`Item "${data.createInboxItem.item.text}" created (id: ${data.createInboxItem.item.id})`)
      setText('')
      setContext('')
      // Refetch the list
      await listItems.execute(INBOX_ITEMS_QUERY, { limit, offset })
    }
  }

  async function handleDelete(id: string) {
    setSuccessMsg(null)
    const data = await deleteItem.execute(DELETE_INBOX_ITEM_MUTATION, { id })
    if (data?.deleteInboxItem?.itemId) {
      setSuccessMsg(`Item deleted (id: ${data.deleteInboxItem.itemId})`)
      // Refetch the list
      await listItems.execute(INBOX_ITEMS_QUERY, { limit, offset })
    }
  }

  async function handleClearAll() {
    if (!window.confirm('Are you sure you want to clear all inbox items? This cannot be undone.')) {
      return
    }
    setSuccessMsg(null)
    const data = await clearInbox.execute(CLEAR_INBOX_MUTATION)
    if (data?.clearInbox) {
      setSuccessMsg(`Cleared inbox: ${data.clearInbox.deletedCount} items deleted`)
      // Refetch the list
      await listItems.execute(INBOX_ITEMS_QUERY, { limit, offset })
    }
  }

  function handlePrev() {
    const newOffset = Math.max(0, offset - limit)
    setOffset(newOffset)
    setSuccessMsg(null)
    listItems.execute(INBOX_ITEMS_QUERY, { limit, offset: newOffset })
  }

  function handleNext() {
    const newOffset = offset + limit
    setOffset(newOffset)
    setSuccessMsg(null)
    listItems.execute(INBOX_ITEMS_QUERY, { limit, offset: newOffset })
  }

  // ---------- Render ----------

  const items = listItems.data?.inboxItems.items ?? []
  const totalCount = listItems.data?.inboxItems.totalCount ?? 0

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold text-gray-800">Inbox</h1>
      <p className="text-sm text-gray-500">
        Quick-capture words and phrases for later review. All operations require authentication.
      </p>

      {/* Create form */}
      <div className="bg-white border border-gray-200 rounded-lg p-4 space-y-3">
        <h3 className="text-sm font-semibold text-gray-700">Add Item</h3>
        <form onSubmit={handleCreate} className="space-y-3">
          <div>
            <label className="block text-xs text-gray-500 mb-1">Text *</label>
            <input
              type="text"
              value={text}
              onChange={(e) => setText(e.target.value)}
              required
              placeholder="Word or phrase..."
              className="w-full border border-gray-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">Context (optional)</label>
            <textarea
              value={context}
              onChange={(e) => setContext(e.target.value)}
              rows={2}
              placeholder="Where did you encounter this word?"
              className="w-full border border-gray-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
            />
          </div>
          <button
            type="submit"
            disabled={createItem.loading || !text.trim()}
            className="bg-blue-600 text-white px-4 py-2 rounded text-sm hover:bg-blue-700 disabled:opacity-50"
          >
            {createItem.loading ? 'Adding...' : 'Add'}
          </button>
        </form>
      </div>

      {/* Success message */}
      {successMsg && (
        <div className="bg-green-50 border border-green-200 text-green-700 text-sm rounded p-3">
          {successMsg}
        </div>
      )}

      {/* GraphQL errors */}
      {listItems.errors && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
          {listItems.errors.map((err, i) => <div key={i}>{err.message}</div>)}
        </div>
      )}
      {createItem.errors && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
          <strong>Create error: </strong>
          {createItem.errors.map((err, i) => <div key={i}>{err.message}</div>)}
        </div>
      )}
      {deleteItem.errors && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
          <strong>Delete error: </strong>
          {deleteItem.errors.map((err, i) => <div key={i}>{err.message}</div>)}
        </div>
      )}
      {clearInbox.errors && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
          <strong>Clear error: </strong>
          {clearInbox.errors.map((err, i) => <div key={i}>{err.message}</div>)}
        </div>
      )}

      {/* Pagination controls */}
      <div className="bg-white border border-gray-200 rounded-lg p-4 space-y-3">
        <div className="flex items-center gap-3">
          <div>
            <label className="block text-xs text-gray-500 mb-1">limit</label>
            <input
              type="number"
              min={1}
              max={100}
              value={limit}
              onChange={(e) => setLimit(Number(e.target.value))}
              className="w-20 border border-gray-300 rounded px-2 py-1 text-sm"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">offset</label>
            <input
              type="number"
              min={0}
              value={offset}
              onChange={(e) => setOffset(Number(e.target.value))}
              className="w-20 border border-gray-300 rounded px-2 py-1 text-sm"
            />
          </div>
          <div className="flex items-end">
            <button
              onClick={handleFetch}
              disabled={listItems.loading}
              className="px-4 py-1 text-sm bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
            >
              {listItems.loading ? 'Loading...' : 'Fetch'}
            </button>
          </div>
          {listItems.data && (
            <span className="text-xs text-gray-500 ml-auto">
              Total: {totalCount}
            </span>
          )}
        </div>

        {listItems.data && totalCount > limit && (
          <div className="flex gap-2">
            <button
              onClick={handlePrev}
              disabled={listItems.loading || offset === 0}
              className="px-3 py-1 text-sm border border-gray-300 rounded hover:bg-gray-50 disabled:opacity-50"
            >
              Prev
            </button>
            <button
              onClick={handleNext}
              disabled={listItems.loading || offset + limit >= totalCount}
              className="px-3 py-1 text-sm border border-gray-300 rounded hover:bg-gray-50 disabled:opacity-50"
            >
              Next
            </button>
          </div>
        )}
      </div>

      {/* Clear All button */}
      {listItems.data && totalCount > 0 && (
        <div>
          <button
            onClick={handleClearAll}
            disabled={clearInbox.loading}
            className="px-4 py-2 text-sm bg-red-600 text-white rounded hover:bg-red-700 disabled:opacity-50"
          >
            {clearInbox.loading ? 'Clearing...' : 'Clear All'}
          </button>
        </div>
      )}

      {/* Items list */}
      {listItems.data && (
        <div className="space-y-2">
          {items.length === 0 ? (
            <div className="text-center py-6">
              <div className="text-gray-400 text-sm">Your inbox is empty.</div>
              <div className="text-gray-400 text-xs mt-1">Add words you encounter for later review using the form above.</div>
            </div>
          ) : (
            items.map((item) => (
              <div key={item.id} className="bg-white border border-gray-200 rounded-lg p-3 flex items-start justify-between">
                <div className="space-y-1">
                  <div className="font-medium text-gray-900">{item.text}</div>
                  {item.context && (
                    <div className="text-sm text-gray-500">{item.context}</div>
                  )}
                  <div className="text-xs text-gray-400">
                    {new Date(item.createdAt).toLocaleDateString()}
                  </div>
                </div>
                <button
                  onClick={() => handleDelete(item.id)}
                  disabled={deleteItem.loading}
                  className="ml-4 px-3 py-1 text-sm bg-red-600 text-white rounded hover:bg-red-700 disabled:opacity-50 shrink-0"
                >
                  {deleteItem.loading ? '...' : 'Delete'}
                </button>
              </div>
            ))
          )}
        </div>
      )}

      {/* Raw panel */}
      <RawPanel raw={lastRaw} />
    </div>
  )
}
