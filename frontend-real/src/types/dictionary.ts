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

export type EntrySortField = 'TEXT' | 'CREATED_AT' | 'UPDATED_AT'
export type SortDirection = 'ASC' | 'DESC'

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

export interface CatalogImage {
  id: UUID
  url: string
  caption?: string
}

export interface UserImage {
  id: UUID
  url: string
  caption?: string
  createdAt: string
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
  catalogImages: CatalogImage[]
  userImages: UserImage[]
  pronunciations: Pronunciation[]
  topics: Topic[]
  createdAt: string
  updatedAt: string
}

export interface PageInfo {
  hasNextPage: boolean
  hasPreviousPage: boolean
  startCursor?: string
  endCursor?: string
}

export interface DictionaryEdge {
  node: DictionaryEntry
  cursor: string
}

export interface DictionaryConnection {
  edges: DictionaryEdge[]
  pageInfo: PageInfo
  totalCount: number
}

export interface DictionaryFilterInput {
  search?: string
  topicId?: UUID
  partOfSpeech?: PartOfSpeech
  sortField?: EntrySortField
  sortDirection?: SortDirection
  first?: number
  after?: string
  limit?: number
  offset?: number
}
