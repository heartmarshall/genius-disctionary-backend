export type UUID = string

export type PartOfSpeech =
  | 'NOUN'
  | 'VERB'
  | 'ADJECTIVE'
  | 'ADVERB'
  | 'PRONOUN'
  | 'PREPOSITION'
  | 'CONJUNCTION'
  | 'INTERJECTION'
  | 'PHRASE'
  | 'IDIOM'
  | 'OTHER'

export type SortBy = 'TEXT' | 'CREATED_AT' | 'UPDATED_AT'
export type SortDir = 'ASC' | 'DESC'

export interface Translation {
  id: UUID
  text: string
}

export interface Example {
  id: UUID
  sentence: string
  translation?: string
}

export interface Sense {
  id: UUID
  definition?: string
  partOfSpeech?: PartOfSpeech
  translations: Translation[]
  examples: Example[]
}

export interface Image {
  id: UUID
  url: string
  caption?: string
}

export interface Pronunciation {
  id: UUID
  audioUrl?: string
  transcription: string
  region?: string
}

export interface Topic {
  id: UUID
  name: string
  description?: string
}

export interface DictionaryEntry {
  id: UUID
  text: string
  textNormalized: string
  notes?: string
  senses: Sense[]
  images: Image[]
  pronunciations: Pronunciation[]
  topics: Topic[]
  createdAt: string
  updatedAt: string
}

export interface WordFilter {
  search?: string
  topicIDs?: UUID[]
  partOfSpeech?: PartOfSpeech
  sortBy?: SortBy
  sortDir?: SortDir
  limit?: number
}
