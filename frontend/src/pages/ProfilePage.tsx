import { useState, useEffect } from 'react'
import { useGraphQL } from '../hooks/useGraphQL'
import { RawPanel } from '../components/RawPanel'

// ---------- GraphQL queries / mutations ----------

const ME_QUERY = `
query Me {
  me {
    id email name avatarUrl oauthProvider createdAt
    settings { newCardsPerDay reviewsPerDay maxIntervalDays timezone }
  }
}`

const UPDATE_SETTINGS_MUTATION = `
mutation UpdateSettings($input: UpdateSettingsInput!) {
  updateSettings(input: $input) {
    settings { newCardsPerDay reviewsPerDay maxIntervalDays timezone }
  }
}`

// ---------- Types ----------

interface UserSettings {
  newCardsPerDay: number
  reviewsPerDay: number
  maxIntervalDays: number
  timezone: string
}

interface MeData {
  me: {
    id: string
    email: string
    name: string | null
    avatarUrl: string | null
    oauthProvider: string
    createdAt: string
    settings: UserSettings
  }
}

interface UpdateSettingsData {
  updateSettings: {
    settings: UserSettings
  }
}

// ---------- Component ----------

export function ProfilePage() {
  const me = useGraphQL<MeData>()
  const updateSettings = useGraphQL<UpdateSettingsData>()

  // Form state
  const [newCardsPerDay, setNewCardsPerDay] = useState<number | ''>('')
  const [reviewsPerDay, setReviewsPerDay] = useState<number | ''>('')
  const [maxIntervalDays, setMaxIntervalDays] = useState<number | ''>('')
  const [timezone, setTimezone] = useState('')

  // Messages
  const [successMsg, setSuccessMsg] = useState<string | null>(null)

  // Track original settings to only send changed fields
  const [originalSettings, setOriginalSettings] = useState<UserSettings | null>(null)

  // Load profile on mount
  useEffect(() => {
    me.execute(ME_QUERY)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // Pre-fill form when data loads
  useEffect(() => {
    if (me.data?.me.settings) {
      const s = me.data.me.settings
      setNewCardsPerDay(s.newCardsPerDay)
      setReviewsPerDay(s.reviewsPerDay)
      setMaxIntervalDays(s.maxIntervalDays)
      setTimezone(s.timezone)
      setOriginalSettings(s)
    }
  }, [me.data])

  // The raw panel shows the most recent operation
  const lastRaw = updateSettings.raw ?? me.raw

  async function handleSaveSettings(e: React.FormEvent) {
    e.preventDefault()
    setSuccessMsg(null)

    // Build input with only changed fields
    const input: Record<string, unknown> = {}

    if (originalSettings) {
      if (newCardsPerDay !== '' && newCardsPerDay !== originalSettings.newCardsPerDay) {
        input.newCardsPerDay = Number(newCardsPerDay)
      }
      if (reviewsPerDay !== '' && reviewsPerDay !== originalSettings.reviewsPerDay) {
        input.reviewsPerDay = Number(reviewsPerDay)
      }
      if (maxIntervalDays !== '' && maxIntervalDays !== originalSettings.maxIntervalDays) {
        input.maxIntervalDays = Number(maxIntervalDays)
      }
      if (timezone !== originalSettings.timezone) {
        input.timezone = timezone
      }
    } else {
      // No original settings — send all fields
      if (newCardsPerDay !== '') input.newCardsPerDay = Number(newCardsPerDay)
      if (reviewsPerDay !== '') input.reviewsPerDay = Number(reviewsPerDay)
      if (maxIntervalDays !== '') input.maxIntervalDays = Number(maxIntervalDays)
      if (timezone.trim()) input.timezone = timezone.trim()
    }

    if (Object.keys(input).length === 0) {
      setSuccessMsg('No changes to save.')
      return
    }

    const data = await updateSettings.execute(UPDATE_SETTINGS_MUTATION, { input })
    if (data?.updateSettings?.settings) {
      const s = data.updateSettings.settings
      setNewCardsPerDay(s.newCardsPerDay)
      setReviewsPerDay(s.reviewsPerDay)
      setMaxIntervalDays(s.maxIntervalDays)
      setTimezone(s.timezone)
      setOriginalSettings(s)
      setSuccessMsg('Settings saved successfully.')
    }
  }

  // ---------- Render ----------

  return (
    <div className="p-6 max-w-3xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold text-gray-800">Profile</h1>
      <p className="text-sm text-gray-500">
        View your profile information and manage SRS settings.
      </p>

      {/* Loading state */}
      {me.loading && (
        <div className="text-gray-500 text-sm">Loading profile...</div>
      )}

      {/* Errors */}
      {me.errors && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
          {me.errors.map((err, i) => <div key={i}>{err.message}</div>)}
        </div>
      )}
      {updateSettings.errors && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
          <strong>Update error: </strong>
          {updateSettings.errors.map((err, i) => <div key={i}>{err.message}</div>)}
        </div>
      )}

      {/* Success message */}
      {successMsg && (
        <div className="bg-green-50 border border-green-200 text-green-700 text-sm rounded p-3">
          {successMsg}
        </div>
      )}

      {/* Section 1: User Profile (read-only) */}
      {me.data?.me && (
        <div className="bg-white border border-gray-200 rounded-lg p-5 space-y-4">
          <h2 className="text-lg font-semibold text-gray-800">User Information</h2>

          <div className="flex items-start gap-4">
            {/* Avatar */}
            {me.data.me.avatarUrl && (
              <img
                src={me.data.me.avatarUrl}
                alt="User avatar"
                className="w-16 h-16 rounded-full border border-gray-200 object-cover"
              />
            )}

            {/* Info fields */}
            <div className="flex-1 grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label className="block text-xs text-gray-500 mb-0.5">ID</label>
                <div className="text-sm font-mono text-gray-700 bg-gray-50 rounded px-2 py-1 break-all">
                  {me.data.me.id}
                </div>
              </div>
              <div>
                <label className="block text-xs text-gray-500 mb-0.5">Email</label>
                <div className="text-sm text-gray-700 bg-gray-50 rounded px-2 py-1">
                  {me.data.me.email}
                </div>
              </div>
              <div>
                <label className="block text-xs text-gray-500 mb-0.5">Name</label>
                <div className="text-sm text-gray-700 bg-gray-50 rounded px-2 py-1">
                  {me.data.me.name ?? '-'}
                </div>
              </div>
              <div>
                <label className="block text-xs text-gray-500 mb-0.5">OAuth Provider</label>
                <div className="text-sm text-gray-700 bg-gray-50 rounded px-2 py-1">
                  {me.data.me.oauthProvider}
                </div>
              </div>
              <div>
                <label className="block text-xs text-gray-500 mb-0.5">Created At</label>
                <div className="text-sm text-gray-700 bg-gray-50 rounded px-2 py-1">
                  {new Date(me.data.me.createdAt).toLocaleString()}
                </div>
              </div>
              {!me.data.me.avatarUrl && (
                <div>
                  <label className="block text-xs text-gray-500 mb-0.5">Avatar</label>
                  <div className="text-sm text-gray-400 bg-gray-50 rounded px-2 py-1">
                    No avatar
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Current settings (read-only display) */}
          <div className="border-t border-gray-100 pt-3">
            <h3 className="text-sm font-semibold text-gray-700 mb-2">Current SRS Settings</h3>
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
              <div>
                <label className="block text-xs text-gray-500 mb-0.5">New Cards/Day</label>
                <div className="text-sm font-medium text-gray-700 bg-gray-50 rounded px-2 py-1">
                  {me.data.me.settings.newCardsPerDay}
                </div>
              </div>
              <div>
                <label className="block text-xs text-gray-500 mb-0.5">Reviews/Day</label>
                <div className="text-sm font-medium text-gray-700 bg-gray-50 rounded px-2 py-1">
                  {me.data.me.settings.reviewsPerDay}
                </div>
              </div>
              <div>
                <label className="block text-xs text-gray-500 mb-0.5">Max Interval (days)</label>
                <div className="text-sm font-medium text-gray-700 bg-gray-50 rounded px-2 py-1">
                  {me.data.me.settings.maxIntervalDays}
                </div>
              </div>
              <div>
                <label className="block text-xs text-gray-500 mb-0.5">Timezone</label>
                <div className="text-sm font-medium text-gray-700 bg-gray-50 rounded px-2 py-1">
                  {me.data.me.settings.timezone}
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Section 2: Settings Form (editable) */}
      {me.data?.me && (
        <div className="bg-white border border-gray-200 rounded-lg p-5 space-y-4">
          <h2 className="text-lg font-semibold text-gray-800">Update Settings</h2>
          <p className="text-xs text-gray-500">
            Only changed fields will be sent to the server.
          </p>

          <form onSubmit={handleSaveSettings} className="space-y-4">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  New Cards Per Day
                </label>
                <input
                  type="number"
                  min={0}
                  value={newCardsPerDay}
                  onChange={(e) => setNewCardsPerDay(e.target.value === '' ? '' : Number(e.target.value))}
                  className="w-full border border-gray-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Reviews Per Day
                </label>
                <input
                  type="number"
                  min={0}
                  value={reviewsPerDay}
                  onChange={(e) => setReviewsPerDay(e.target.value === '' ? '' : Number(e.target.value))}
                  className="w-full border border-gray-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Max Interval (days)
                </label>
                <input
                  type="number"
                  min={1}
                  value={maxIntervalDays}
                  onChange={(e) => setMaxIntervalDays(e.target.value === '' ? '' : Number(e.target.value))}
                  className="w-full border border-gray-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Timezone
                </label>
                <input
                  type="text"
                  value={timezone}
                  onChange={(e) => setTimezone(e.target.value)}
                  placeholder="e.g. Europe/Moscow"
                  className="w-full border border-gray-300 rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
                />
              </div>
            </div>

            <button
              type="submit"
              disabled={updateSettings.loading}
              className="bg-blue-600 text-white px-5 py-2 rounded text-sm font-medium hover:bg-blue-700 disabled:opacity-50"
            >
              {updateSettings.loading ? 'Saving...' : 'Save Settings'}
            </button>
          </form>
        </div>
      )}

      {/* Section 3: Raw Panel */}
      <RawPanel raw={lastRaw} />
    </div>
  )
}
