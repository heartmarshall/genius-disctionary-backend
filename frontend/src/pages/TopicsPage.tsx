import { useState, useEffect } from 'react'
import { useGraphQL } from '../hooks/useGraphQL'
import { useAuth } from '../auth/useAuth'
import { RawPanel } from '../components/RawPanel'

// ---------- GraphQL queries / mutations ----------

const TOPICS_QUERY = `
query Topics {
  topics { id name description entryCount createdAt updatedAt }
}`

const CREATE_TOPIC_MUTATION = `
mutation CreateTopic($input: CreateTopicInput!) {
  createTopic(input: $input) { topic { id name description entryCount } }
}`

const UPDATE_TOPIC_MUTATION = `
mutation UpdateTopic($input: UpdateTopicInput!) {
  updateTopic(input: $input) { topic { id name description } }
}`

const DELETE_TOPIC_MUTATION = `
mutation DeleteTopic($id: UUID!) {
  deleteTopic(id: $id) { topicId }
}`

const LINK_ENTRY_MUTATION = `
mutation LinkEntry($input: LinkEntryInput!) {
  linkEntryToTopic(input: $input) { success }
}`

const UNLINK_ENTRY_MUTATION = `
mutation UnlinkEntry($input: UnlinkEntryInput!) {
  unlinkEntryFromTopic(input: $input) { success }
}`

const BATCH_LINK_MUTATION = `
mutation BatchLink($input: BatchLinkEntriesInput!) {
  batchLinkEntriesToTopic(input: $input) { linked skipped }
}`

// ---------- Types ----------

interface Topic {
  id: string
  name: string
  description: string | null
  entryCount: number
  createdAt: string
  updatedAt: string
}

interface TopicsData {
  topics: Topic[]
}

interface CreateTopicData {
  createTopic: {
    topic: { id: string; name: string; description: string | null; entryCount: number }
  }
}

interface UpdateTopicData {
  updateTopic: {
    topic: { id: string; name: string; description: string | null }
  }
}

interface DeleteTopicData {
  deleteTopic: { topicId: string }
}

interface LinkEntryData {
  linkEntryToTopic: { success: boolean }
}

interface UnlinkEntryData {
  unlinkEntryFromTopic: { success: boolean }
}

interface BatchLinkData {
  batchLinkEntriesToTopic: { linked: number; skipped: number }
}

// ---------- Component ----------

export function TopicsPage() {
  const { isAuthenticated } = useAuth()

  // Create form state
  const [createName, setCreateName] = useState('')
  const [createDescription, setCreateDescription] = useState('')

  // Edit state
  const [editingTopicId, setEditingTopicId] = useState<string | null>(null)
  const [editName, setEditName] = useState('')
  const [editDescription, setEditDescription] = useState('')

  // Expanded topic (for entry linking)
  const [expandedTopicId, setExpandedTopicId] = useState<string | null>(null)

  // Link/unlink form state per topic
  const [linkEntryId, setLinkEntryId] = useState('')
  const [unlinkEntryId, setUnlinkEntryId] = useState('')
  const [batchEntryIds, setBatchEntryIds] = useState('')

  // Success message
  const [successMsg, setSuccessMsg] = useState<string | null>(null)

  // GraphQL hooks
  const topicsQuery = useGraphQL<TopicsData>()
  const createTopic = useGraphQL<CreateTopicData>()
  const updateTopic = useGraphQL<UpdateTopicData>()
  const deleteTopic = useGraphQL<DeleteTopicData>()
  const linkEntry = useGraphQL<LinkEntryData>()
  const unlinkEntry = useGraphQL<UnlinkEntryData>()
  const batchLink = useGraphQL<BatchLinkData>()

  // Raw panel shows the most recent operation
  const lastRaw = batchLink.raw ?? unlinkEntry.raw ?? linkEntry.raw
    ?? deleteTopic.raw ?? updateTopic.raw ?? createTopic.raw ?? topicsQuery.raw

  // Auto-load topics on mount
  useEffect(() => {
    if (isAuthenticated) {
      topicsQuery.execute(TOPICS_QUERY)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // ---------- Handlers ----------

  async function handleFetchTopics() {
    setSuccessMsg(null)
    await topicsQuery.execute(TOPICS_QUERY)
  }

  async function handleCreateTopic(e: React.FormEvent) {
    e.preventDefault()
    setSuccessMsg(null)
    const input: Record<string, unknown> = { name: createName.trim() }
    if (createDescription.trim()) input.description = createDescription.trim()

    const data = await createTopic.execute(CREATE_TOPIC_MUTATION, { input })
    if (data?.createTopic?.topic) {
      setSuccessMsg(`Topic "${data.createTopic.topic.name}" created (id: ${data.createTopic.topic.id})`)
      setCreateName('')
      setCreateDescription('')
      await topicsQuery.execute(TOPICS_QUERY)
    }
  }

  function startEdit(topic: Topic) {
    setEditingTopicId(topic.id)
    setEditName(topic.name)
    setEditDescription(topic.description ?? '')
  }

  function cancelEdit() {
    setEditingTopicId(null)
    setEditName('')
    setEditDescription('')
  }

  async function handleSaveEdit(topicId: string) {
    setSuccessMsg(null)
    const input: Record<string, unknown> = { topicId }
    if (editName.trim()) input.name = editName.trim()
    if (editDescription.trim()) {
      input.description = editDescription.trim()
    } else {
      input.description = null
    }

    const data = await updateTopic.execute(UPDATE_TOPIC_MUTATION, { input })
    if (data?.updateTopic?.topic) {
      setSuccessMsg(`Topic "${data.updateTopic.topic.name}" updated`)
      setEditingTopicId(null)
      await topicsQuery.execute(TOPICS_QUERY)
    }
  }

  async function handleDeleteTopic(topicId: string, topicName: string) {
    if (!window.confirm(`Delete topic "${topicName}"? This cannot be undone.`)) return
    setSuccessMsg(null)
    const data = await deleteTopic.execute(DELETE_TOPIC_MUTATION, { id: topicId })
    if (data?.deleteTopic?.topicId) {
      setSuccessMsg(`Topic "${topicName}" deleted`)
      await topicsQuery.execute(TOPICS_QUERY)
    }
  }

  function toggleExpand(topicId: string) {
    if (expandedTopicId === topicId) {
      setExpandedTopicId(null)
    } else {
      setExpandedTopicId(topicId)
      setLinkEntryId('')
      setUnlinkEntryId('')
      setBatchEntryIds('')
    }
  }

  async function handleLinkEntry(topicId: string) {
    if (!linkEntryId.trim()) return
    setSuccessMsg(null)
    const data = await linkEntry.execute(LINK_ENTRY_MUTATION, {
      input: { topicId, entryId: linkEntryId.trim() },
    })
    if (data?.linkEntryToTopic?.success) {
      setSuccessMsg('Entry linked successfully')
      setLinkEntryId('')
      await topicsQuery.execute(TOPICS_QUERY)
    }
  }

  async function handleUnlinkEntry(topicId: string) {
    if (!unlinkEntryId.trim()) return
    setSuccessMsg(null)
    const data = await unlinkEntry.execute(UNLINK_ENTRY_MUTATION, {
      input: { topicId, entryId: unlinkEntryId.trim() },
    })
    if (data?.unlinkEntryFromTopic?.success) {
      setSuccessMsg('Entry unlinked successfully')
      setUnlinkEntryId('')
      await topicsQuery.execute(TOPICS_QUERY)
    }
  }

  async function handleBatchLink(topicId: string) {
    const entryIds = batchEntryIds
      .split('\n')
      .map((line) => line.trim())
      .filter(Boolean)
    if (entryIds.length === 0) return
    setSuccessMsg(null)
    const data = await batchLink.execute(BATCH_LINK_MUTATION, {
      input: { topicId, entryIds },
    })
    if (data?.batchLinkEntriesToTopic) {
      const r = data.batchLinkEntriesToTopic
      setSuccessMsg(`Batch link: ${r.linked} linked, ${r.skipped} skipped`)
      setBatchEntryIds('')
      await topicsQuery.execute(TOPICS_QUERY)
    }
  }

  // ---------- Render ----------

  const topics = topicsQuery.data?.topics ?? []

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold text-gray-800">Topics</h1>
      <p className="text-sm text-gray-500">
        Manage topics and link dictionary entries. All operations require authentication.
      </p>

      {/* Fetch button */}
      <button
        onClick={handleFetchTopics}
        disabled={topicsQuery.loading}
        className="bg-blue-600 text-white px-4 py-1.5 rounded text-sm hover:bg-blue-700 disabled:opacity-50"
      >
        {topicsQuery.loading ? 'Loading...' : 'Fetch Topics'}
      </button>

      {/* Success message */}
      {successMsg && (
        <div className="bg-green-50 border border-green-200 text-green-700 text-sm rounded p-3">
          {successMsg}
        </div>
      )}

      {/* Errors */}
      {topicsQuery.errors && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
          {topicsQuery.errors.map((err, i) => <div key={i}>{err.message}</div>)}
        </div>
      )}
      {createTopic.errors && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
          <strong>Create error: </strong>
          {createTopic.errors.map((err, i) => <div key={i}>{err.message}</div>)}
        </div>
      )}
      {updateTopic.errors && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
          <strong>Update error: </strong>
          {updateTopic.errors.map((err, i) => <div key={i}>{err.message}</div>)}
        </div>
      )}
      {deleteTopic.errors && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
          <strong>Delete error: </strong>
          {deleteTopic.errors.map((err, i) => <div key={i}>{err.message}</div>)}
        </div>
      )}
      {linkEntry.errors && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
          <strong>Link error: </strong>
          {linkEntry.errors.map((err, i) => <div key={i}>{err.message}</div>)}
        </div>
      )}
      {unlinkEntry.errors && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
          <strong>Unlink error: </strong>
          {unlinkEntry.errors.map((err, i) => <div key={i}>{err.message}</div>)}
        </div>
      )}
      {batchLink.errors && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
          <strong>Batch link error: </strong>
          {batchLink.errors.map((err, i) => <div key={i}>{err.message}</div>)}
        </div>
      )}

      {/* ===== Create Topic Form ===== */}
      <div className="bg-white border border-gray-200 rounded-lg p-5 shadow-sm space-y-3">
        <h2 className="text-lg font-bold text-gray-800">Create Topic</h2>
        <form onSubmit={handleCreateTopic} className="space-y-3">
          <div>
            <label className="block text-sm font-semibold text-gray-700 mb-1">Name *</label>
            <input
              type="text"
              value={createName}
              onChange={(e) => setCreateName(e.target.value)}
              required
              placeholder="Topic name..."
              className="w-full border border-gray-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
            />
          </div>
          <div>
            <label className="block text-sm font-semibold text-gray-700 mb-1">Description</label>
            <textarea
              value={createDescription}
              onChange={(e) => setCreateDescription(e.target.value)}
              rows={2}
              placeholder="Optional description..."
              className="w-full border border-gray-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
            />
          </div>
          <button
            type="submit"
            disabled={createTopic.loading || !createName.trim()}
            className="bg-green-600 text-white px-4 py-2 rounded text-sm hover:bg-green-700 disabled:opacity-50"
          >
            {createTopic.loading ? 'Creating...' : 'Create'}
          </button>
        </form>
      </div>

      {/* ===== Topics List ===== */}
      {topicsQuery.data && (
        <div className="space-y-4">
          <h2 className="text-lg font-bold text-gray-800">
            Topics ({topics.length})
          </h2>

          {topics.length === 0 ? (
            <div className="text-center py-6">
              <div className="text-gray-400 text-sm">No topics yet.</div>
              <div className="text-gray-400 text-xs mt-1">Create your first topic above to organize your vocabulary.</div>
            </div>
          ) : (
            topics.map((topic) => {
              const isEditing = editingTopicId === topic.id
              const isExpanded = expandedTopicId === topic.id

              return (
                <div key={topic.id} className="bg-white border border-gray-200 rounded-lg shadow-sm">
                  {/* Topic header */}
                  <div className="p-4">
                    {isEditing ? (
                      /* Edit mode */
                      <div className="space-y-3">
                        <div>
                          <label className="block text-xs text-gray-500 mb-1">Name</label>
                          <input
                            type="text"
                            value={editName}
                            onChange={(e) => setEditName(e.target.value)}
                            className="w-full border border-gray-300 rounded px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
                          />
                        </div>
                        <div>
                          <label className="block text-xs text-gray-500 mb-1">Description</label>
                          <input
                            type="text"
                            value={editDescription}
                            onChange={(e) => setEditDescription(e.target.value)}
                            placeholder="Optional description..."
                            className="w-full border border-gray-300 rounded px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
                          />
                        </div>
                        <div className="flex gap-2">
                          <button
                            onClick={() => handleSaveEdit(topic.id)}
                            disabled={updateTopic.loading || !editName.trim()}
                            className="text-xs px-3 py-1 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
                          >
                            {updateTopic.loading ? 'Saving...' : 'Save'}
                          </button>
                          <button
                            onClick={cancelEdit}
                            className="text-xs px-3 py-1 bg-gray-200 text-gray-700 rounded hover:bg-gray-300"
                          >
                            Cancel
                          </button>
                        </div>
                      </div>
                    ) : (
                      /* Display mode */
                      <div className="flex items-start justify-between">
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 mb-1">
                            <span className="font-bold text-gray-900">{topic.name}</span>
                            <span className="text-xs bg-indigo-100 text-indigo-700 px-2 py-0.5 rounded-full">
                              {topic.entryCount} {topic.entryCount === 1 ? 'entry' : 'entries'}
                            </span>
                          </div>
                          {topic.description && (
                            <p className="text-sm text-gray-600 mb-1">{topic.description}</p>
                          )}
                          <div className="text-xs text-gray-400">
                            Created: {new Date(topic.createdAt).toLocaleDateString()} | Updated: {new Date(topic.updatedAt).toLocaleDateString()}
                          </div>
                        </div>
                        <div className="flex items-center gap-2 ml-3 shrink-0">
                          <button
                            onClick={() => toggleExpand(topic.id)}
                            className="text-xs px-3 py-1 bg-indigo-50 text-indigo-700 border border-indigo-200 rounded hover:bg-indigo-100"
                          >
                            {isExpanded ? 'Collapse' : 'Entries'}
                          </button>
                          <button
                            onClick={() => startEdit(topic)}
                            className="text-xs px-3 py-1 bg-blue-50 text-blue-700 border border-blue-200 rounded hover:bg-blue-100"
                          >
                            Edit
                          </button>
                          <button
                            onClick={() => handleDeleteTopic(topic.id, topic.name)}
                            disabled={deleteTopic.loading}
                            className="text-xs px-3 py-1 bg-red-50 text-red-700 border border-red-200 rounded hover:bg-red-100 disabled:opacity-50"
                          >
                            Delete
                          </button>
                        </div>
                      </div>
                    )}
                  </div>

                  {/* Expandable entry linking section */}
                  {isExpanded && !isEditing && (
                    <div className="border-t border-gray-100 p-4 bg-gray-50 space-y-4">
                      {/* Link Entry */}
                      <div className="space-y-2">
                        <h4 className="text-sm font-semibold text-gray-700">Link Entry</h4>
                        <div className="flex gap-2">
                          <input
                            type="text"
                            value={linkEntryId}
                            onChange={(e) => setLinkEntryId(e.target.value)}
                            placeholder="Entry UUID..."
                            className="flex-1 border border-gray-300 rounded px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-green-400"
                          />
                          <button
                            onClick={() => handleLinkEntry(topic.id)}
                            disabled={linkEntry.loading || !linkEntryId.trim()}
                            className="text-xs px-3 py-1.5 bg-green-600 text-white rounded hover:bg-green-700 disabled:opacity-50"
                          >
                            {linkEntry.loading ? 'Linking...' : 'Link'}
                          </button>
                        </div>
                      </div>

                      {/* Unlink Entry */}
                      <div className="space-y-2">
                        <h4 className="text-sm font-semibold text-gray-700">Unlink Entry</h4>
                        <div className="flex gap-2">
                          <input
                            type="text"
                            value={unlinkEntryId}
                            onChange={(e) => setUnlinkEntryId(e.target.value)}
                            placeholder="Entry UUID..."
                            className="flex-1 border border-gray-300 rounded px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-red-400"
                          />
                          <button
                            onClick={() => handleUnlinkEntry(topic.id)}
                            disabled={unlinkEntry.loading || !unlinkEntryId.trim()}
                            className="text-xs px-3 py-1.5 bg-red-600 text-white rounded hover:bg-red-700 disabled:opacity-50"
                          >
                            {unlinkEntry.loading ? 'Unlinking...' : 'Unlink'}
                          </button>
                        </div>
                      </div>

                      {/* Batch Link */}
                      <div className="space-y-2">
                        <h4 className="text-sm font-semibold text-gray-700">Batch Link Entries</h4>
                        <textarea
                          value={batchEntryIds}
                          onChange={(e) => setBatchEntryIds(e.target.value)}
                          rows={4}
                          placeholder="Paste entry UUIDs, one per line..."
                          className="w-full border border-gray-300 rounded px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-400"
                        />
                        <button
                          onClick={() => handleBatchLink(topic.id)}
                          disabled={batchLink.loading || !batchEntryIds.trim()}
                          className="text-xs px-3 py-1.5 bg-indigo-600 text-white rounded hover:bg-indigo-700 disabled:opacity-50"
                        >
                          {batchLink.loading ? 'Linking...' : 'Batch Link'}
                        </button>
                      </div>
                    </div>
                  )}
                </div>
              )
            })
          )}
        </div>
      )}

      {/* Raw panel */}
      <RawPanel raw={lastRaw} />
    </div>
  )
}
