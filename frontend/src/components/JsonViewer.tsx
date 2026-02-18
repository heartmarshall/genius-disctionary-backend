interface Props {
  data: unknown
  maxHeight?: string
}

export function JsonViewer({ data, maxHeight = '400px' }: Props) {
  const json = JSON.stringify(data, null, 2)

  return (
    <pre
      className="bg-gray-900 text-green-400 text-xs p-3 rounded overflow-auto font-mono"
      style={{ maxHeight }}
    >
      {json}
    </pre>
  )
}
