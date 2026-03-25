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

export type CardState = 'NEW' | 'LEARNING' | 'REVIEW' | 'RELEARNING'
export type ReviewGrade = 'AGAIN' | 'HARD' | 'GOOD' | 'EASY'

export interface Card {
  id: UUID
  entryId: UUID
  state: CardState
  due: string
  lastReview?: string
  reps: Int
  lapses: Int
  scheduledDays: Int
  createdAt: string
  updatedAt: string
}

type Int = number

export interface GradeCounts {
  again: number
  hard: number
  good: number
  easy: number
}

export interface CardStats {
  totalReviews: number
  accuracy: number
  currentState: CardState
  scheduledDays: number
  gradeDistribution?: GradeCounts
}

export interface ReviewLog {
  id: UUID
  cardId: UUID
  grade: ReviewGrade
  reviewedAt: string
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
  card?: Card
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

// ─── Reference Catalog Types ──────────────────────────────────────────────

export interface RefTranslation {
  id: UUID
  text: string
  sourceSlug: string
}

export interface RefExample {
  id: UUID
  sentence: string
  translation?: string
  sourceSlug: string
}

export interface RefSense {
  id: UUID
  definition?: string
  partOfSpeech?: PartOfSpeech
  cefrLevel?: string
  notes?: string
  sourceSlug: string
  position: number
  translations: RefTranslation[]
  examples: RefExample[]
}

export interface RefPronunciation {
  id: UUID
  transcription: string
  audioUrl?: string
  region?: string
}

export interface RefEntry {
  id: UUID
  text: string
  textNormalized: string
  frequencyRank?: number
  cefrLevel?: string
  isCoreLexicon: boolean
  senses: RefSense[]
  pronunciations: RefPronunciation[]
}

// ─── Filter Types ─────────────────────────────────────────────────────────

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
