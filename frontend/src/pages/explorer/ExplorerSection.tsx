import { useState } from 'react'
import { OperationForm } from './OperationForm'
import type { OperationGroup } from './operations'

interface Props {
  group: OperationGroup
}

export function ExplorerSection({ group }: Props) {
  const [openOp, setOpenOp] = useState<string | null>(null)

  function toggle(name: string) {
    setOpenOp((prev) => (prev === name ? null : name))
  }

  return (
    <div className="space-y-2">
      <div className="text-sm text-gray-500 mb-3">
        {group.operations.length} operation{group.operations.length !== 1 ? 's' : ''}
      </div>

      {group.operations.map((op) => {
        const isOpen = openOp === op.name
        return (
          <div key={op.name} className="border border-gray-200 rounded-lg overflow-hidden">
            <button
              onClick={() => toggle(op.name)}
              className="w-full text-left px-4 py-3 bg-white hover:bg-gray-50 flex items-center justify-between"
            >
              <div className="flex items-center gap-2">
                <span
                  className={`text-xs font-bold px-1.5 py-0.5 rounded ${
                    op.type === 'query'
                      ? 'bg-blue-100 text-blue-700'
                      : 'bg-red-100 text-red-700'
                  }`}
                >
                  {op.type === 'query' ? 'Q' : 'M'}
                </span>
                <span className="text-sm font-medium text-gray-900">{op.name}</span>
                <span className="text-xs text-gray-400 hidden sm:inline">{op.description}</span>
              </div>
              <span className="text-gray-400 text-sm">{isOpen ? '▼' : '▶'}</span>
            </button>
            {isOpen && (
              <div className="px-4 py-4 bg-gray-50 border-t border-gray-200">
                <OperationForm operation={op} />
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}
