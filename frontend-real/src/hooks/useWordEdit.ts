import { useMutation } from '@apollo/client'
import { UPDATE_WORD, GET_DICTIONARY } from '@/graphql/queries/dictionary'

export interface SenseInput {
  definition?: string
  partOfSpeech?: string
  translations: { text: string }[]
  examples?: { sentence: string; translation?: string }[]
}

export interface ImageInput {
  url: string
  caption?: string
}

export interface PronunciationInput {
  transcription: string
  audioUrl?: string
  region?: string
}

export interface UpdateWordInput {
  text?: string
  notes?: string
  senses?: SenseInput[]
  images?: ImageInput[]
  pronunciations?: PronunciationInput[]
  topicIDs?: string[]
}

export function useWordEdit() {
  const [mutate, { loading }] = useMutation(UPDATE_WORD, {
    refetchQueries: [{ query: GET_DICTIONARY }],
  })

  const updateWord = async (id: string, input: UpdateWordInput) => {
    await mutate({ variables: { id, input } })
  }

  return { updateWord, loading }
}
