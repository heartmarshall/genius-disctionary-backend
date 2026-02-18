import { useState } from 'react'
import { JsonViewer } from './JsonViewer'

interface Props {
  raw: { query: string; variables: unknown; response: unknown; status: number } | null
}

export function RawPanel({ raw }: Props) {
  const [open, setOpen] = useState(false)

  if (!raw) return null

  return (
    <div className="mt-4 border border-gray-300 rounded">
      <button
        onClick={() => setOpen(!open)}
        className="w-full text-left px-3 py-2 bg-gray-100 hover:bg-gray-200 text-sm font-mono flex justify-between items-center"
      >
        <span>Raw Request/Response (HTTP {raw.status})</span>
        <span>{open ? '▼' : '▶'}</span>
      </button>
      {open && (
        <div className="p-3 space-y-3">
          <div>
            <h4 className="text-xs font-bold mb-1">Query:</h4>
            <pre className="bg-gray-800 text-blue-300 text-xs p-2 rounded overflow-auto max-h-48 font-mono">
              {raw.query}
            </pre>
          </div>
          {raw.variables != null && (
            <div>
              <h4 className="text-xs font-bold mb-1">Variables:</h4>
              <JsonViewer data={raw.variables} maxHeight="150px" />
            </div>
          )}
          <div>
            <h4 className="text-xs font-bold mb-1">Response:</h4>
            <JsonViewer data={raw.response} maxHeight="300px" />
          </div>
          <button
            onClick={() => {
              const curl = `curl -X POST ${window.location.origin}/query \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer <TOKEN>" \\
  -d '${JSON.stringify({ query: raw.query, variables: raw.variables })}'`
              navigator.clipboard.writeText(curl)
            }}
            className="text-xs bg-gray-700 text-white px-2 py-1 rounded hover:bg-gray-600"
          >
            Copy as cURL
          </button>
        </div>
      )}
    </div>
  )
}
