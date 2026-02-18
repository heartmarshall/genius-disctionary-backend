import { useState } from 'react'
import { useGraphQL } from '../../hooks/useGraphQL'
import { RawPanel } from '../../components/RawPanel'
import { JsonViewer } from '../../components/JsonViewer'
import type { Operation, OperationField } from './operations'

interface Props {
  operation: Operation
}

export function OperationForm({ operation }: Props) {
  const gql = useGraphQL()
  const [values, setValues] = useState<Record<string, string>>(() => {
    const initial: Record<string, string> = {}
    for (const field of operation.fields) {
      initial[field.name] = field.defaultValue ?? ''
    }
    return initial
  })

  function setValue(name: string, value: string) {
    setValues((prev) => ({ ...prev, [name]: value }))
  }

  function buildVariables(): Record<string, unknown> {
    const vars: Record<string, unknown> = {}

    for (const field of operation.fields) {
      const raw = values[field.name]?.trim() ?? ''

      if (field.type === 'boolean') {
        // Checkboxes: always include the value
        vars[field.name] = raw === 'true'
        continue
      }

      if (raw === '') continue

      switch (field.type) {
        case 'number':
          vars[field.name] = parseInt(raw, 10)
          break
        case 'json':
          vars[field.name] = JSON.parse(raw)
          break
        case 'uuid[]': {
          const ids = raw.split('\n').map((s) => s.trim()).filter((s) => s.length > 0)
          vars[field.name] = ids
          break
        }
        case 'string[]': {
          const strs = raw.split('\n').map((s) => s.trim()).filter((s) => s.length > 0)
          vars[field.name] = strs
          break
        }
        default:
          vars[field.name] = raw
          break
      }
    }

    // Heuristic: if the query contains `$input:`, wrap fields in { input: {...} }
    const usesInputWrapper = operation.query.includes('$input:')
    if (usesInputWrapper) {
      return { input: vars }
    }
    return vars
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    gql.reset()
    try {
      const variables = buildVariables()
      await gql.execute(operation.query, variables)
    } catch {
      // JSON parse error etc. — handled by gql.errors
    }
  }

  function renderField(field: OperationField) {
    const key = field.name
    const value = values[key] ?? ''

    const label = (
      <label className="block text-xs font-medium text-gray-600 mb-1">
        {field.name}
        {field.required && <span className="text-red-500 ml-0.5">*</span>}
        <span className="ml-1 text-gray-400">({field.type})</span>
      </label>
    )

    if (field.type === 'boolean') {
      return (
        <div key={key} className="flex items-center gap-2">
          <input
            type="checkbox"
            checked={value === 'true'}
            onChange={(e) => setValue(key, e.target.checked ? 'true' : 'false')}
            className="h-4 w-4"
          />
          <label className="text-xs font-medium text-gray-600">
            {field.name}
            <span className="ml-1 text-gray-400">(boolean)</span>
          </label>
        </div>
      )
    }

    if (field.type === 'enum') {
      return (
        <div key={key}>
          {label}
          <select
            value={value}
            onChange={(e) => setValue(key, e.target.value)}
            className="w-full border border-gray-300 rounded px-2 py-1.5 text-sm bg-white focus:outline-none focus:ring-2 focus:ring-blue-400"
          >
            <option value="">-- select --</option>
            {field.enumValues?.map((v) => (
              <option key={v} value={v}>{v}</option>
            ))}
          </select>
        </div>
      )
    }

    if (field.type === 'json' || field.type === 'uuid[]' || field.type === 'string[]') {
      return (
        <div key={key}>
          {label}
          <textarea
            value={value}
            onChange={(e) => setValue(key, e.target.value)}
            rows={3}
            placeholder={field.placeholder ?? (field.type === 'uuid[]' ? 'One UUID per line' : field.type === 'string[]' ? 'One string per line' : '')}
            className="w-full border border-gray-300 rounded px-2 py-1.5 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-blue-400"
          />
        </div>
      )
    }

    if (field.type === 'number') {
      return (
        <div key={key}>
          {label}
          <input
            type="number"
            value={value}
            onChange={(e) => setValue(key, e.target.value)}
            placeholder={field.placeholder ?? ''}
            className="w-full border border-gray-300 rounded px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
          />
        </div>
      )
    }

    // string, uuid
    return (
      <div key={key}>
        {label}
        <input
          type="text"
          value={value}
          onChange={(e) => setValue(key, e.target.value)}
          placeholder={field.placeholder ?? (field.type === 'uuid' ? '00000000-0000-0000-0000-000000000000' : '')}
          className={`w-full border border-gray-300 rounded px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400 ${field.type === 'uuid' ? 'font-mono' : ''}`}
        />
      </div>
    )
  }

  // Check for validation field errors
  const validationFields = gql.errors
    ?.flatMap((err) => err.extensions?.fields ?? []) ?? []

  return (
    <div className="space-y-4">
      {/* Badge */}
      <div className="flex items-center gap-2">
        <span
          className={`text-xs font-bold px-2 py-0.5 rounded ${
            operation.type === 'query'
              ? 'bg-blue-100 text-blue-700'
              : 'bg-red-100 text-red-700'
          }`}
        >
          {operation.type.toUpperCase()}
        </span>
        <span className="text-sm text-gray-500">{operation.description}</span>
      </div>

      {/* Form */}
      <form onSubmit={handleSubmit} className="space-y-3">
        {operation.fields.length > 0 && (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {operation.fields.map(renderField)}
          </div>
        )}

        <button
          type="submit"
          disabled={gql.loading}
          className={`px-4 py-1.5 text-sm text-white rounded disabled:opacity-50 ${
            operation.type === 'query'
              ? 'bg-blue-600 hover:bg-blue-700'
              : 'bg-red-600 hover:bg-red-700'
          }`}
        >
          {gql.loading ? 'Executing...' : 'Execute'}
        </button>
      </form>

      {/* Loading */}
      {gql.loading && (
        <div className="text-sm text-gray-500">Loading...</div>
      )}

      {/* Errors */}
      {gql.errors && (
        <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3 space-y-1">
          {gql.errors.map((err, i) => (
            <div key={i}>
              <span className="font-medium">{err.extensions?.code ?? 'ERROR'}:</span> {err.message}
            </div>
          ))}
          {validationFields.length > 0 && (
            <div className="mt-2 border-t border-red-200 pt-2">
              <div className="text-xs font-semibold text-red-800 mb-1">Validation Errors:</div>
              {validationFields.map((vf, i) => (
                <div key={i} className="text-xs">
                  <span className="font-mono bg-red-100 px-1 rounded">{vf.field}</span>: {vf.message}
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Success result */}
      {gql.data != null && !gql.loading && (
        <div className="space-y-2">
          <div className="text-xs font-semibold text-green-700">Result:</div>
          <JsonViewer data={gql.data} maxHeight="400px" />
        </div>
      )}

      {/* Raw panel */}
      <RawPanel raw={gql.raw} />
    </div>
  )
}
