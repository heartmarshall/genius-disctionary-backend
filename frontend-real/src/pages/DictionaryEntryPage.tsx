import { useParams } from 'react-router'

function DictionaryEntryPage() {
  const { id } = useParams<{ id: string }>()

  return (
    <div>
      <h1 className="font-heading text-2xl mb-md">Word Details</h1>
      <p className="text-text-secondary">Entry ID: {id}</p>
    </div>
  )
}

export default DictionaryEntryPage
