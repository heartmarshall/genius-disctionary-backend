import { useParams } from 'react-router-dom'

export function DictionaryEntryPage() {
  const { id } = useParams()
  return <h1 className="text-3xl font-bold">Dictionary Entry: {id}</h1>
}
