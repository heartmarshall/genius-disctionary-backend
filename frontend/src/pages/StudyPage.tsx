import { useState, useRef, useCallback, useEffect } from 'react'
import { useGraphQL } from '../hooks/useGraphQL'
import { RawPanel } from '../components/RawPanel'
import { useAuth } from '../auth/useAuth'

// ---------- GraphQL queries / mutations ----------

const DASHBOARD_QUERY = `
query Dashboard {
  dashboard {
    dueCount newCount reviewedToday streak overdueCount
    statusCounts { new learning review mastered }
    activeSession { id status startedAt finishedAt
      result { totalReviews gradeCounts { again hard good easy } averageDurationMs }
    }
  }
}`

const STUDY_QUEUE_QUERY = `
query StudyQueue($limit: Int) {
  studyQueue(limit: $limit) {
    id text
    senses { id definition partOfSpeech translations { id text } }
    card { id status nextReviewAt intervalDays easeFactor }
  }
}`

const CARD_HISTORY_QUERY = `
query CardHistory($input: GetCardHistoryInput!) {
  cardHistory(input: $input) { id cardId grade durationMs reviewedAt }
}`

const CARD_STATS_QUERY = `
query CardStats($cardId: UUID!) {
  cardStats(cardId: $cardId) {
    totalReviews averageDurationMs accuracy
    gradeDistribution { again hard good easy }
  }
}`

const REVIEW_CARD_MUTATION = `
mutation ReviewCard($input: ReviewCardInput!) {
  reviewCard(input: $input) { card { id status nextReviewAt intervalDays easeFactor } }
}`

const UNDO_REVIEW_MUTATION = `
mutation UndoReview($cardId: UUID!) {
  undoReview(cardId: $cardId) { card { id status nextReviewAt intervalDays easeFactor } }
}`

const BATCH_CREATE_CARDS_MUTATION = `
mutation BatchCreateCards($entryIds: [UUID!]!) {
  batchCreateCards(entryIds: $entryIds) { createdCount skippedCount errors { entryId message } }
}`

const START_SESSION_MUTATION = `
mutation StartSession {
  startStudySession { session { id status startedAt } }
}`

const FINISH_SESSION_MUTATION = `
mutation FinishSession($input: FinishSessionInput!) {
  finishStudySession(input: $input) {
    session { id status finishedAt
      result { totalReviews gradeCounts { again hard good easy } averageDurationMs }
    }
  }
}`

const ABANDON_SESSION_MUTATION = `
mutation AbandonSession {
  abandonStudySession { success }
}`

// ---------- Types ----------

interface StatusCounts {
  new: number
  learning: number
  review: number
  mastered: number
}

interface GradeCounts {
  again: number
  hard: number
  good: number
  easy: number
}

interface SessionResult {
  totalReviews: number
  gradeCounts: GradeCounts
  averageDurationMs: number
}

interface ActiveSession {
  id: string
  status: string
  startedAt: string
  finishedAt: string | null
  result: SessionResult | null
}

interface DashboardData {
  dashboard: {
    dueCount: number
    newCount: number
    reviewedToday: number
    streak: number
    overdueCount: number
    statusCounts: StatusCounts
    activeSession: ActiveSession | null
  }
}

interface CardInfo {
  id: string
  status: string
  nextReviewAt: string | null
  intervalDays: number
  easeFactor: number
}

interface QueueEntry {
  id: string
  text: string
  senses: {
    id: string
    definition: string
    partOfSpeech: string
    translations: { id: string; text: string }[]
  }[]
  card: CardInfo | null
}

interface StudyQueueData {
  studyQueue: QueueEntry[]
}

interface ReviewLog {
  id: string
  cardId: string
  grade: string
  durationMs: number | null
  reviewedAt: string
}

interface CardHistoryData {
  cardHistory: ReviewLog[]
}

interface CardStatsData {
  cardStats: {
    totalReviews: number
    averageDurationMs: number
    accuracy: number
    gradeDistribution: GradeCounts
  }
}

interface ReviewCardData {
  reviewCard: { card: CardInfo }
}

interface UndoReviewData {
  undoReview: { card: CardInfo }
}

interface BatchCreateCardsData {
  batchCreateCards: {
    createdCount: number
    skippedCount: number
    errors: { entryId: string; message: string }[]
  }
}

interface StartSessionData {
  startStudySession: {
    session: { id: string; status: string; startedAt: string }
  }
}

interface FinishSessionData {
  finishStudySession: {
    session: {
      id: string
      status: string
      finishedAt: string
      result: SessionResult | null
    }
  }
}

interface AbandonSessionData {
  abandonStudySession: { success: boolean }
}

// ---------- Component ----------

export function StudyPage() {
  const { isAuthenticated } = useAuth()

  // Dashboard
  const dashboard = useGraphQL<DashboardData>()

  // Session
  const startSession = useGraphQL<StartSessionData>()
  const finishSession = useGraphQL<FinishSessionData>()
  const abandonSession = useGraphQL<AbandonSessionData>()
  const [sessionId, setSessionId] = useState('')
  const [finishedResult, setFinishedResult] = useState<FinishSessionData['finishStudySession']['session'] | null>(null)

  // Queue
  const studyQueue = useGraphQL<StudyQueueData>()
  const [queueLimit, setQueueLimit] = useState(10)
  const [queue, setQueue] = useState<QueueEntry[]>([])

  // Review mode
  const [reviewMode, setReviewMode] = useState(false)
  const [reviewIndex, setReviewIndex] = useState(0)
  const [revealed, setRevealed] = useState(false)
  const [reviewSummary, setReviewSummary] = useState<GradeCounts | null>(null)
  const cardShownAt = useRef<number>(0)
  const reviewCard = useGraphQL<ReviewCardData>()
  const [lastReviewResult, setLastReviewResult] = useState<CardInfo | null>(null)

  // Card Inspector
  const [inspectCardId, setInspectCardId] = useState('')
  const [historyLimit, setHistoryLimit] = useState(20)
  const [historyOffset, setHistoryOffset] = useState(0)
  const cardHistory = useGraphQL<CardHistoryData>()
  const cardStats = useGraphQL<CardStatsData>()

  // Undo
  const [undoCardId, setUndoCardId] = useState('')
  const undoReview = useGraphQL<UndoReviewData>()

  // Batch Create
  const [batchEntryIds, setBatchEntryIds] = useState('')
  const batchCreate = useGraphQL<BatchCreateCardsData>()

  // Messages
  const [successMsg, setSuccessMsg] = useState<string | null>(null)

  // Raw panel: most recent operation
  const lastRaw = reviewCard.raw ?? undoReview.raw ?? batchCreate.raw
    ?? finishSession.raw ?? startSession.raw ?? abandonSession.raw
    ?? cardHistory.raw ?? cardStats.raw ?? studyQueue.raw ?? dashboard.raw

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

  // ---------- Handlers ----------

  async function handleLoadDashboard() {
    setSuccessMsg(null)
    await dashboard.execute(DASHBOARD_QUERY)
  }

  async function handleStartSession() {
    setSuccessMsg(null)
    setFinishedResult(null)
    const data = await startSession.execute(START_SESSION_MUTATION)
    if (data?.startStudySession?.session) {
      const s = data.startStudySession.session
      setSessionId(s.id)
      setSuccessMsg(`Session started (id: ${s.id}, status: ${s.status})`)
    }
  }

  async function handleFinishSession() {
    if (!sessionId.trim()) return
    setSuccessMsg(null)
    const data = await finishSession.execute(FINISH_SESSION_MUTATION, {
      input: { sessionId: sessionId.trim() },
    })
    if (data?.finishStudySession?.session) {
      setFinishedResult(data.finishStudySession.session)
      setSuccessMsg('Session finished!')
    }
  }

  async function handleAbandonSession() {
    setSuccessMsg(null)
    const data = await abandonSession.execute(ABANDON_SESSION_MUTATION)
    if (data?.abandonStudySession?.success) {
      setSessionId('')
      setSuccessMsg('Session abandoned.')
    }
  }

  async function handleLoadQueue() {
    setSuccessMsg(null)
    setReviewMode(false)
    setReviewSummary(null)
    const data = await studyQueue.execute(STUDY_QUEUE_QUERY, { limit: queueLimit })
    if (data?.studyQueue) {
      setQueue(data.studyQueue)
    }
  }

  const handleStartReview = useCallback(() => {
    if (queue.length === 0) return
    setReviewMode(true)
    setReviewIndex(0)
    setRevealed(false)
    setLastReviewResult(null)
    setReviewSummary(null)
    cardShownAt.current = Date.now()
  }, [queue.length])

  async function handleGrade(grade: string) {
    const entry = queue[reviewIndex]
    if (!entry?.card) return
    const durationMs = Date.now() - cardShownAt.current
    setLastReviewResult(null)

    const data = await reviewCard.execute(REVIEW_CARD_MUTATION, {
      input: { cardId: entry.card.id, grade, durationMs },
    })

    if (data?.reviewCard?.card) {
      setLastReviewResult(data.reviewCard.card)

      // Update running summary
      setReviewSummary((prev) => {
        const counts = prev ?? { again: 0, hard: 0, good: 0, easy: 0 }
        const key = grade.toLowerCase() as keyof GradeCounts
        return { ...counts, [key]: counts[key] + 1 }
      })

      // Brief pause to show result, then move on
      setTimeout(() => {
        setLastReviewResult(null)
        const nextIdx = reviewIndex + 1
        if (nextIdx >= queue.length) {
          setReviewMode(false)
          // summary is already set
        } else {
          setReviewIndex(nextIdx)
          setRevealed(false)
          cardShownAt.current = Date.now()
        }
      }, 800)
    }
  }

  async function handleGetHistory() {
    if (!inspectCardId.trim()) return
    setSuccessMsg(null)
    await cardHistory.execute(CARD_HISTORY_QUERY, {
      input: { cardId: inspectCardId.trim(), limit: historyLimit, offset: historyOffset },
    })
  }

  async function handleGetStats() {
    if (!inspectCardId.trim()) return
    setSuccessMsg(null)
    await cardStats.execute(CARD_STATS_QUERY, { cardId: inspectCardId.trim() })
  }

  async function handleUndoReview() {
    if (!undoCardId.trim()) return
    setSuccessMsg(null)
    const data = await undoReview.execute(UNDO_REVIEW_MUTATION, { cardId: undoCardId.trim() })
    if (data?.undoReview?.card) {
      const c = data.undoReview.card
      setSuccessMsg(`Undo successful. Card ${c.id} now: status=${c.status}, interval=${c.intervalDays}d`)
    }
  }

  async function handleBatchCreate() {
    const ids = batchEntryIds
      .split('\n')
      .map((s) => s.trim())
      .filter((s) => s.length > 0)
    if (ids.length === 0) return
    setSuccessMsg(null)
    const data = await batchCreate.execute(BATCH_CREATE_CARDS_MUTATION, { entryIds: ids })
    if (data?.batchCreateCards) {
      const r = data.batchCreateCards
      setSuccessMsg(
        `Batch create: ${r.createdCount} created, ${r.skippedCount} skipped` +
          (r.errors.length > 0 ? `, ${r.errors.length} errors` : ''),
      )
    }
  }

  // ---------- Render helpers ----------

  function renderDashboard() {
    const d = dashboard.data?.dashboard
    return (
      <div className="bg-white border border-gray-200 rounded-lg p-4 space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-bold text-gray-800">Dashboard</h2>
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
        </div>

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
          <div className="space-y-3">
            {/* Stats row */}
            <div className="grid grid-cols-5 gap-3">
              <div className="bg-blue-50 border border-blue-200 rounded p-3 text-center">
                <div className="text-2xl font-bold text-blue-700">{d.dueCount}</div>
                <div className="text-xs text-blue-600">Due</div>
              </div>
              <div className="bg-green-50 border border-green-200 rounded p-3 text-center">
                <div className="text-2xl font-bold text-green-700">{d.newCount}</div>
                <div className="text-xs text-green-600">New</div>
              </div>
              <div className="bg-purple-50 border border-purple-200 rounded p-3 text-center">
                <div className="text-2xl font-bold text-purple-700">{d.reviewedToday}</div>
                <div className="text-xs text-purple-600">Reviewed Today</div>
              </div>
              <div className="bg-orange-50 border border-orange-200 rounded p-3 text-center">
                <div className="text-2xl font-bold text-orange-700">{d.streak}</div>
                <div className="text-xs text-orange-600">Streak</div>
              </div>
              <div className="bg-red-50 border border-red-200 rounded p-3 text-center">
                <div className="text-2xl font-bold text-red-700">{d.overdueCount}</div>
                <div className="text-xs text-red-600">Overdue</div>
              </div>
            </div>

            {/* Status counts */}
            <div className="bg-gray-50 border border-gray-200 rounded p-3">
              <h3 className="text-xs font-semibold text-gray-500 mb-2">Status Breakdown</h3>
              <div className="flex gap-4 text-sm">
                <span className="text-blue-700">New: {d.statusCounts.new}</span>
                <span className="text-yellow-700">Learning: {d.statusCounts.learning}</span>
                <span className="text-purple-700">Review: {d.statusCounts.review}</span>
                <span className="text-green-700">Mastered: {d.statusCounts.mastered}</span>
              </div>
            </div>

            {/* Active session */}
            {d.activeSession && (
              <div className="bg-yellow-50 border border-yellow-200 rounded p-3">
                <h3 className="text-xs font-semibold text-yellow-700 mb-1">Active Session</h3>
                <div className="text-sm text-gray-700">
                  <span className="font-mono text-xs">{d.activeSession.id}</span>
                  <span className="ml-2">Status: {d.activeSession.status}</span>
                  <span className="ml-2">Started: {new Date(d.activeSession.startedAt).toLocaleString()}</span>
                </div>
                {d.activeSession.result && (
                  <div className="text-xs text-gray-500 mt-1">
                    Result: {d.activeSession.result.totalReviews} reviews,
                    avg {d.activeSession.result.averageDurationMs}ms,
                    grades: A={d.activeSession.result.gradeCounts.again}
                    H={d.activeSession.result.gradeCounts.hard}
                    G={d.activeSession.result.gradeCounts.good}
                    E={d.activeSession.result.gradeCounts.easy}
                  </div>
                )}
              </div>
            )}
          </div>
        )}
      </div>
    )
  }

  function renderSessionControls() {
    return (
      <div className="bg-white border border-gray-200 rounded-lg p-4 space-y-3">
        <h2 className="text-lg font-bold text-gray-800">Session Controls</h2>
        <div className="flex flex-wrap gap-3 items-end">
          <button
            onClick={handleStartSession}
            disabled={startSession.loading}
            className="px-4 py-1.5 text-sm bg-green-600 text-white rounded hover:bg-green-700 disabled:opacity-50"
          >
            {startSession.loading ? 'Starting...' : 'Start Session'}
          </button>

          <div className="flex items-end gap-2">
            <div>
              <label className="block text-xs text-gray-500 mb-1">Session ID</label>
              <input
                type="text"
                value={sessionId}
                onChange={(e) => setSessionId(e.target.value)}
                placeholder="UUID..."
                className="w-72 border border-gray-300 rounded px-2 py-1.5 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-blue-400"
              />
            </div>
            <button
              onClick={handleFinishSession}
              disabled={finishSession.loading || !sessionId.trim()}
              className="px-4 py-1.5 text-sm bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
            >
              {finishSession.loading ? 'Finishing...' : 'Finish Session'}
            </button>
          </div>

          <button
            onClick={handleAbandonSession}
            disabled={abandonSession.loading}
            className="px-4 py-1.5 text-sm bg-red-600 text-white rounded hover:bg-red-700 disabled:opacity-50"
          >
            {abandonSession.loading ? 'Abandoning...' : 'Abandon Session'}
          </button>
        </div>

        {startSession.errors && (
          <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
            {startSession.errors.map((err, i) => <div key={i}>{err.message}</div>)}
          </div>
        )}
        {finishSession.errors && (
          <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
            {finishSession.errors.map((err, i) => <div key={i}>{err.message}</div>)}
          </div>
        )}
        {abandonSession.errors && (
          <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
            {abandonSession.errors.map((err, i) => <div key={i}>{err.message}</div>)}
          </div>
        )}

        {/* Finished session result */}
        {finishedResult && (
          <div className="bg-green-50 border border-green-200 rounded p-3">
            <h3 className="text-sm font-semibold text-green-800 mb-1">Session Finished</h3>
            <div className="text-sm text-gray-700">
              <div>ID: <span className="font-mono text-xs">{finishedResult.id}</span></div>
              <div>Status: {finishedResult.status}</div>
              <div>Finished: {new Date(finishedResult.finishedAt).toLocaleString()}</div>
              {finishedResult.result && (
                <div className="mt-1 text-xs text-gray-600">
                  Total Reviews: {finishedResult.result.totalReviews} |
                  Avg Duration: {finishedResult.result.averageDurationMs}ms |
                  Grades: Again={finishedResult.result.gradeCounts.again},
                  Hard={finishedResult.result.gradeCounts.hard},
                  Good={finishedResult.result.gradeCounts.good},
                  Easy={finishedResult.result.gradeCounts.easy}
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    )
  }

  function renderStudyQueue() {
    return (
      <div className="bg-white border border-gray-200 rounded-lg p-4 space-y-3">
        <h2 className="text-lg font-bold text-gray-800">Study Queue</h2>
        <div className="flex items-end gap-3">
          <div>
            <label className="block text-xs text-gray-500 mb-1">Limit</label>
            <select
              value={queueLimit}
              onChange={(e) => setQueueLimit(Number(e.target.value))}
              className="border border-gray-300 rounded px-2 py-1.5 text-sm bg-white"
            >
              <option value={5}>5</option>
              <option value={10}>10</option>
              <option value={20}>20</option>
            </select>
          </div>
          <button
            onClick={handleLoadQueue}
            disabled={studyQueue.loading}
            className="px-4 py-1.5 text-sm bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
          >
            {studyQueue.loading ? 'Loading...' : 'Load Queue'}
          </button>
          {queue.length > 0 && !reviewMode && (
            <button
              onClick={handleStartReview}
              className="px-4 py-1.5 text-sm bg-purple-600 text-white rounded hover:bg-purple-700"
            >
              Start Review ({queue.length} cards)
            </button>
          )}
        </div>

        {studyQueue.errors && (
          <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
            {studyQueue.errors.map((err, i) => <div key={i}>{err.message}</div>)}
          </div>
        )}

        {/* Queue list */}
        {queue.length > 0 && !reviewMode && (
          <div className="space-y-2">
            <h3 className="text-sm font-medium text-gray-500">{queue.length} entries in queue</h3>
            {queue.map((entry, idx) => (
              <div key={entry.id} className="bg-gray-50 border border-gray-200 rounded p-3 flex items-center justify-between">
                <div>
                  <span className="text-xs text-gray-400 mr-2">#{idx + 1}</span>
                  <span className="font-medium text-gray-900">{entry.text}</span>
                  {entry.card && (
                    <span className="ml-2 text-xs text-gray-500">
                      [{entry.card.status} | interval: {entry.card.intervalDays}d | EF: {entry.card.easeFactor.toFixed(2)}]
                    </span>
                  )}
                </div>
                <div className="text-xs text-gray-400">
                  {entry.senses.length} sense(s)
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    )
  }

  function renderReviewMode() {
    if (!reviewMode || queue.length === 0) return null
    const entry = queue[reviewIndex]
    if (!entry) return null

    const statusColors: Record<string, string> = {
      NEW: 'bg-blue-100 text-blue-800',
      LEARNING: 'bg-yellow-100 text-yellow-800',
      REVIEW: 'bg-purple-100 text-purple-800',
      MASTERED: 'bg-green-100 text-green-800',
    }

    return (
      <div className="border-2 border-purple-300 rounded-lg p-6 bg-purple-50 space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-bold text-purple-900">
            Review Mode ({reviewIndex + 1} / {queue.length})
          </h2>
          <button
            onClick={() => setReviewMode(false)}
            className="text-xs text-gray-500 hover:text-gray-700"
          >
            Exit Review
          </button>
        </div>

        {/* Card display */}
        <div className="bg-white border border-gray-200 rounded-lg p-6 text-center min-h-[200px] flex flex-col items-center justify-center">
          <div className="text-3xl font-bold text-gray-900 mb-2">{entry.text}</div>
          {entry.card && (
            <span className={`text-xs px-2 py-0.5 rounded ${statusColors[entry.card.status] ?? 'bg-gray-100 text-gray-600'}`}>
              {entry.card.status}
            </span>
          )}

          {!revealed ? (
            <button
              onClick={() => setRevealed(true)}
              className="mt-6 px-6 py-2 bg-gray-800 text-white rounded-lg hover:bg-gray-700 text-sm"
            >
              Show Answer
            </button>
          ) : (
            <div className="mt-4 w-full text-left space-y-2">
              <hr className="border-gray-200" />
              {entry.senses.map((sense) => (
                <div key={sense.id} className="py-1">
                  <span className="text-sm font-medium text-indigo-700">{sense.partOfSpeech}</span>
                  <span className="ml-2 text-sm text-gray-700">{sense.definition}</span>
                  {sense.translations.length > 0 && (
                    <div className="text-xs text-gray-500 ml-4">
                      {sense.translations.map((t) => t.text).join(', ')}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Grade buttons */}
        {revealed && !lastReviewResult && (
          <div className="flex justify-center gap-3">
            <button
              onClick={() => handleGrade('AGAIN')}
              disabled={reviewCard.loading}
              className="px-6 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 disabled:opacity-50 font-medium"
            >
              Again
            </button>
            <button
              onClick={() => handleGrade('HARD')}
              disabled={reviewCard.loading}
              className="px-6 py-2 bg-orange-500 text-white rounded-lg hover:bg-orange-600 disabled:opacity-50 font-medium"
            >
              Hard
            </button>
            <button
              onClick={() => handleGrade('GOOD')}
              disabled={reviewCard.loading}
              className="px-6 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:opacity-50 font-medium"
            >
              Good
            </button>
            <button
              onClick={() => handleGrade('EASY')}
              disabled={reviewCard.loading}
              className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 font-medium"
            >
              Easy
            </button>
          </div>
        )}

        {reviewCard.loading && (
          <div className="text-center text-sm text-gray-500">Submitting review...</div>
        )}

        {reviewCard.errors && (
          <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
            {reviewCard.errors.map((err, i) => <div key={i}>{err.message}</div>)}
          </div>
        )}

        {/* Brief result display */}
        {lastReviewResult && (
          <div className="bg-green-50 border border-green-200 rounded p-3 text-center text-sm text-green-800">
            Updated: {lastReviewResult.status} | Next review: {lastReviewResult.nextReviewAt ? new Date(lastReviewResult.nextReviewAt).toLocaleDateString() : 'N/A'} |
            Interval: {lastReviewResult.intervalDays}d | EF: {lastReviewResult.easeFactor.toFixed(2)}
          </div>
        )}
      </div>
    )
  }

  function renderReviewSummary() {
    if (!reviewSummary || reviewMode) return null
    const total = reviewSummary.again + reviewSummary.hard + reviewSummary.good + reviewSummary.easy
    if (total === 0) return null

    return (
      <div className="bg-purple-50 border border-purple-200 rounded-lg p-4 space-y-2">
        <h2 className="text-lg font-bold text-purple-900">Review Summary</h2>
        <div className="grid grid-cols-4 gap-3 text-center">
          <div className="bg-red-50 border border-red-200 rounded p-2">
            <div className="text-xl font-bold text-red-700">{reviewSummary.again}</div>
            <div className="text-xs text-red-600">Again</div>
          </div>
          <div className="bg-orange-50 border border-orange-200 rounded p-2">
            <div className="text-xl font-bold text-orange-700">{reviewSummary.hard}</div>
            <div className="text-xs text-orange-600">Hard</div>
          </div>
          <div className="bg-green-50 border border-green-200 rounded p-2">
            <div className="text-xl font-bold text-green-700">{reviewSummary.good}</div>
            <div className="text-xs text-green-600">Good</div>
          </div>
          <div className="bg-blue-50 border border-blue-200 rounded p-2">
            <div className="text-xl font-bold text-blue-700">{reviewSummary.easy}</div>
            <div className="text-xs text-blue-600">Easy</div>
          </div>
        </div>
        <div className="text-sm text-gray-600 text-center">Total: {total} reviews</div>
        <button
          onClick={() => setReviewSummary(null)}
          className="text-xs text-gray-500 hover:text-gray-700"
        >
          Dismiss
        </button>
      </div>
    )
  }

  function renderCardInspector() {
    return (
      <div className="bg-white border border-gray-200 rounded-lg p-4 space-y-3">
        <h2 className="text-lg font-bold text-gray-800">Card Inspector</h2>
        <div className="flex items-end gap-3">
          <div className="flex-1">
            <label className="block text-xs text-gray-500 mb-1">Card ID (UUID)</label>
            <input
              type="text"
              value={inspectCardId}
              onChange={(e) => setInspectCardId(e.target.value)}
              placeholder="Enter card UUID..."
              className="w-full border border-gray-300 rounded px-2 py-1.5 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-blue-400"
            />
          </div>
          <div className="flex items-center gap-2">
            <div>
              <label className="block text-xs text-gray-500 mb-1">Limit</label>
              <input
                type="number"
                min={1}
                max={100}
                value={historyLimit}
                onChange={(e) => setHistoryLimit(Number(e.target.value))}
                className="w-16 border border-gray-300 rounded px-2 py-1.5 text-sm"
              />
            </div>
            <div>
              <label className="block text-xs text-gray-500 mb-1">Offset</label>
              <input
                type="number"
                min={0}
                value={historyOffset}
                onChange={(e) => setHistoryOffset(Number(e.target.value))}
                className="w-16 border border-gray-300 rounded px-2 py-1.5 text-sm"
              />
            </div>
          </div>
          <button
            onClick={handleGetHistory}
            disabled={cardHistory.loading || !inspectCardId.trim()}
            className="px-4 py-1.5 text-sm bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
          >
            {cardHistory.loading ? 'Loading...' : 'Get History'}
          </button>
          <button
            onClick={handleGetStats}
            disabled={cardStats.loading || !inspectCardId.trim()}
            className="px-4 py-1.5 text-sm bg-indigo-600 text-white rounded hover:bg-indigo-700 disabled:opacity-50"
          >
            {cardStats.loading ? 'Loading...' : 'Get Stats'}
          </button>
        </div>

        {cardHistory.errors && (
          <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
            {cardHistory.errors.map((err, i) => <div key={i}>{err.message}</div>)}
          </div>
        )}
        {cardStats.errors && (
          <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
            {cardStats.errors.map((err, i) => <div key={i}>{err.message}</div>)}
          </div>
        )}

        {/* History table */}
        {cardHistory.data?.cardHistory && (
          <div>
            <h3 className="text-sm font-semibold text-gray-700 mb-2">
              History ({cardHistory.data.cardHistory.length} records)
            </h3>
            {cardHistory.data.cardHistory.length === 0 ? (
              <div className="text-gray-500 text-sm">No history found.</div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm border border-gray-200 rounded">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="px-3 py-2 text-left">Grade</th>
                      <th className="px-3 py-2 text-left">Duration (ms)</th>
                      <th className="px-3 py-2 text-left">Reviewed At</th>
                    </tr>
                  </thead>
                  <tbody>
                    {cardHistory.data.cardHistory.map((log) => {
                      const gradeColors: Record<string, string> = {
                        AGAIN: 'text-red-700 bg-red-50',
                        HARD: 'text-orange-700 bg-orange-50',
                        GOOD: 'text-green-700 bg-green-50',
                        EASY: 'text-blue-700 bg-blue-50',
                      }
                      return (
                        <tr key={log.id} className="border-t border-gray-100">
                          <td className="px-3 py-2">
                            <span className={`text-xs px-2 py-0.5 rounded font-medium ${gradeColors[log.grade] ?? 'text-gray-700'}`}>
                              {log.grade}
                            </span>
                          </td>
                          <td className="px-3 py-2 text-gray-600">{log.durationMs ?? '-'}</td>
                          <td className="px-3 py-2 text-gray-500 text-xs">{new Date(log.reviewedAt).toLocaleString()}</td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}

        {/* Stats */}
        {cardStats.data?.cardStats && (
          <div className="bg-indigo-50 border border-indigo-200 rounded p-3">
            <h3 className="text-sm font-semibold text-indigo-800 mb-2">Card Stats</h3>
            <div className="grid grid-cols-3 gap-3 text-sm">
              <div>
                <span className="text-gray-500">Total Reviews:</span>{' '}
                <span className="font-medium">{cardStats.data.cardStats.totalReviews}</span>
              </div>
              <div>
                <span className="text-gray-500">Avg Duration:</span>{' '}
                <span className="font-medium">{cardStats.data.cardStats.averageDurationMs}ms</span>
              </div>
              <div>
                <span className="text-gray-500">Accuracy:</span>{' '}
                <span className="font-medium">{(cardStats.data.cardStats.accuracy * 100).toFixed(1)}%</span>
              </div>
            </div>
            <div className="mt-2 text-xs text-gray-600">
              Grade Distribution:
              Again={cardStats.data.cardStats.gradeDistribution.again},
              Hard={cardStats.data.cardStats.gradeDistribution.hard},
              Good={cardStats.data.cardStats.gradeDistribution.good},
              Easy={cardStats.data.cardStats.gradeDistribution.easy}
            </div>
          </div>
        )}
      </div>
    )
  }

  function renderUndoAndBatch() {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* Undo */}
        <div className="bg-white border border-gray-200 rounded-lg p-4 space-y-3">
          <h2 className="text-lg font-bold text-gray-800">Undo Last Review</h2>
          <div className="flex items-end gap-2">
            <div className="flex-1">
              <label className="block text-xs text-gray-500 mb-1">Card ID (UUID)</label>
              <input
                type="text"
                value={undoCardId}
                onChange={(e) => setUndoCardId(e.target.value)}
                placeholder="Enter card UUID..."
                className="w-full border border-gray-300 rounded px-2 py-1.5 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-blue-400"
              />
            </div>
            <button
              onClick={handleUndoReview}
              disabled={undoReview.loading || !undoCardId.trim()}
              className="px-4 py-1.5 text-sm bg-orange-600 text-white rounded hover:bg-orange-700 disabled:opacity-50"
            >
              {undoReview.loading ? 'Undoing...' : 'Undo Review'}
            </button>
          </div>
          {undoReview.errors && (
            <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
              {undoReview.errors.map((err, i) => <div key={i}>{err.message}</div>)}
            </div>
          )}
        </div>

        {/* Batch Create */}
        <div className="bg-white border border-gray-200 rounded-lg p-4 space-y-3">
          <h2 className="text-lg font-bold text-gray-800">Batch Create Cards</h2>
          <div>
            <label className="block text-xs text-gray-500 mb-1">Entry IDs (one UUID per line)</label>
            <textarea
              value={batchEntryIds}
              onChange={(e) => setBatchEntryIds(e.target.value)}
              rows={4}
              placeholder={"entry-uuid-1\nentry-uuid-2\nentry-uuid-3"}
              className="w-full border border-gray-300 rounded px-2 py-1.5 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-blue-400"
            />
          </div>
          <button
            onClick={handleBatchCreate}
            disabled={batchCreate.loading || !batchEntryIds.trim()}
            className="px-4 py-1.5 text-sm bg-green-600 text-white rounded hover:bg-green-700 disabled:opacity-50"
          >
            {batchCreate.loading ? 'Creating...' : 'Batch Create Cards'}
          </button>
          {batchCreate.errors && (
            <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
              {batchCreate.errors.map((err, i) => <div key={i}>{err.message}</div>)}
            </div>
          )}
          {batchCreate.data?.batchCreateCards && (
            <div className="bg-green-50 border border-green-200 text-green-700 text-sm rounded p-3">
              Created: {batchCreate.data.batchCreateCards.createdCount},
              Skipped: {batchCreate.data.batchCreateCards.skippedCount}
              {batchCreate.data.batchCreateCards.errors.length > 0 && (
                <div className="mt-1 text-xs">
                  {batchCreate.data.batchCreateCards.errors.map((err, i) => (
                    <div key={i}>Entry {err.entryId}: {err.message}</div>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    )
  }

  // ---------- Main render ----------

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold text-gray-800">Study</h1>
      <p className="text-sm text-gray-500">
        Spaced repetition study flow. Load dashboard, start sessions, review cards, inspect history.
      </p>

      {/* Success message */}
      {successMsg && (
        <div className="bg-green-50 border border-green-200 text-green-700 text-sm rounded p-3">
          {successMsg}
        </div>
      )}

      {renderDashboard()}
      {renderSessionControls()}
      {renderStudyQueue()}
      {renderReviewMode()}
      {renderReviewSummary()}
      {renderCardInspector()}
      {renderUndoAndBatch()}

      {/* Raw panel */}
      <RawPanel raw={lastRaw} />
    </div>
  )
}
