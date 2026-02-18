import { useState } from 'react'
import { ExplorerSection } from './explorer/ExplorerSection'
import { operationGroups } from './explorer/operations'

export function ExplorerPage() {
  const [activeTab, setActiveTab] = useState(0)
  const activeGroup = operationGroups[activeTab]

  const totalOps = operationGroups.reduce((sum, g) => sum + g.operations.length, 0)

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-4">
      <div>
        <h1 className="text-2xl font-bold text-gray-800">API Explorer</h1>
        <p className="text-sm text-gray-500 mt-1">
          Raw access to every GraphQL operation. {totalOps} operations across {operationGroups.length} groups.
        </p>
      </div>

      {/* Tab bar */}
      <div className="flex gap-1 border-b border-gray-200">
        {operationGroups.map((group, idx) => (
          <button
            key={group.name}
            onClick={() => setActiveTab(idx)}
            className={`px-4 py-2 text-sm font-medium rounded-t transition-colors ${
              idx === activeTab
                ? 'bg-white border border-gray-200 border-b-white text-gray-900 -mb-px'
                : 'text-gray-500 hover:text-gray-700 hover:bg-gray-100'
            }`}
          >
            {group.name}
            <span className="ml-1.5 text-xs text-gray-400">({group.operations.length})</span>
          </button>
        ))}
      </div>

      {/* Active tab content */}
      {activeGroup && <ExplorerSection group={activeGroup} />}
    </div>
  )
}
