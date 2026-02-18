// ---------- Types ----------

export interface OperationField {
  name: string
  type: 'string' | 'number' | 'boolean' | 'uuid' | 'json' | 'enum' | 'uuid[]' | 'string[]'
  required?: boolean
  enumValues?: string[]
  placeholder?: string
  defaultValue?: string
}

export interface Operation {
  name: string
  type: 'query' | 'mutation'
  description: string
  query: string
  fields: OperationField[]
}

export interface OperationGroup {
  name: string
  operations: Operation[]
}

// ============================================================
//  Part-of-speech enum values (reused across multiple ops)
// ============================================================

const PART_OF_SPEECH_VALUES = [
  'NOUN', 'VERB', 'ADJECTIVE', 'ADVERB', 'PRONOUN',
  'PREPOSITION', 'CONJUNCTION', 'INTERJECTION', 'PHRASE', 'IDIOM', 'OTHER',
]

// ============================================================
//  GROUP 1 — RefCatalog
// ============================================================

const refCatalogOperations: Operation[] = [
  {
    name: 'searchCatalog',
    type: 'query',
    description: 'Search the reference catalog (autocomplete). No auth required.',
    query: `query SearchCatalog($query: String!, $limit: Int) {
  searchCatalog(query: $query, limit: $limit) {
    id text textNormalized
    senses { id definition partOfSpeech cefrLevel position
      translations { id text } examples { id sentence translation } }
    pronunciations { id transcription audioUrl region }
    images { id url caption }
  }
}`,
    fields: [
      { name: 'query', type: 'string', required: true, placeholder: 'hello' },
      { name: 'limit', type: 'number', placeholder: '10' },
    ],
  },
  {
    name: 'previewRefEntry',
    type: 'query',
    description: 'Full preview of a word from the catalog. No auth required.',
    query: `query PreviewRefEntry($text: String!) {
  previewRefEntry(text: $text) {
    id text textNormalized
    senses { id definition partOfSpeech cefrLevel position
      translations { id text } examples { id sentence translation } }
    pronunciations { id transcription audioUrl region }
    images { id url caption }
  }
}`,
    fields: [
      { name: 'text', type: 'string', required: true, placeholder: 'hello' },
    ],
  },
]

// ============================================================
//  GROUP 2 — Dictionary
// ============================================================

const dictionaryOperations: Operation[] = [
  {
    name: 'dictionary',
    type: 'query',
    description: 'Search/filter user dictionary with cursor or offset pagination.',
    query: `query Dictionary($input: DictionaryFilterInput!) {
  dictionary(input: $input) {
    edges { cursor node { id text textNormalized notes createdAt updatedAt deletedAt
      senses { id definition partOfSpeech }
      card { id status nextReviewAt } topics { id name } } }
    pageInfo { hasNextPage hasPreviousPage startCursor endCursor }
    totalCount
  }
}`,
    fields: [
      { name: 'search', type: 'string', placeholder: 'word or phrase' },
      { name: 'hasCard', type: 'boolean' },
      { name: 'partOfSpeech', type: 'enum', enumValues: PART_OF_SPEECH_VALUES },
      { name: 'topicId', type: 'uuid' },
      { name: 'status', type: 'enum', enumValues: ['NEW', 'LEARNING', 'REVIEW', 'MASTERED'] },
      { name: 'sortField', type: 'enum', enumValues: ['TEXT', 'CREATED_AT', 'UPDATED_AT'] },
      { name: 'sortDirection', type: 'enum', enumValues: ['ASC', 'DESC'] },
      { name: 'first', type: 'number', placeholder: '10' },
      { name: 'after', type: 'string', placeholder: 'cursor string' },
      { name: 'limit', type: 'number', placeholder: '10' },
      { name: 'offset', type: 'number', placeholder: '0' },
    ],
  },
  {
    name: 'dictionaryEntry',
    type: 'query',
    description: 'Get a single dictionary entry by ID with all nested data.',
    query: `query DictionaryEntry($id: UUID!) {
  dictionaryEntry(id: $id) {
    id text textNormalized notes createdAt updatedAt
    senses { id definition partOfSpeech cefrLevel sourceSlug position
      translations { id text sourceSlug position }
      examples { id sentence translation sourceSlug position } }
    pronunciations { id transcription audioUrl region }
    catalogImages { id url caption }
    userImages { id url caption createdAt }
    card { id status nextReviewAt intervalDays easeFactor createdAt updatedAt }
    topics { id name }
  }
}`,
    fields: [
      { name: 'id', type: 'uuid', required: true },
    ],
  },
  {
    name: 'deletedEntries',
    type: 'query',
    description: 'List soft-deleted entries (trash bin).',
    query: `query DeletedEntries($limit: Int, $offset: Int) {
  deletedEntries(limit: $limit, offset: $offset) {
    entries { id text notes deletedAt } totalCount
  }
}`,
    fields: [
      { name: 'limit', type: 'number', placeholder: '20' },
      { name: 'offset', type: 'number', placeholder: '0' },
    ],
  },
  {
    name: 'exportEntries',
    type: 'query',
    description: 'Export entire dictionary in structured format.',
    query: `query ExportEntries {
  exportEntries {
    exportedAt items { text notes cardStatus createdAt
      senses { definition partOfSpeech translations examples { sentence translation } } }
  }
}`,
    fields: [],
  },
  {
    name: 'createEntryFromCatalog',
    type: 'mutation',
    description: 'Create a dictionary entry from a reference catalog entry.',
    query: `mutation CreateEntryFromCatalog($input: CreateEntryFromCatalogInput!) {
  createEntryFromCatalog(input: $input) { entry { id text } }
}`,
    fields: [
      { name: 'refEntryId', type: 'uuid', required: true },
      { name: 'senseIds', type: 'json', required: true, placeholder: '["uuid1","uuid2"]' },
      { name: 'notes', type: 'string' },
      { name: 'createCard', type: 'boolean' },
    ],
  },
  {
    name: 'createEntryCustom',
    type: 'mutation',
    description: 'Create a custom dictionary entry (without catalog).',
    query: `mutation CreateEntryCustom($input: CreateEntryCustomInput!) {
  createEntryCustom(input: $input) { entry { id text notes } }
}`,
    fields: [
      { name: 'text', type: 'string', required: true, placeholder: 'hello' },
      { name: 'senses', type: 'json', required: true, placeholder: '[{"definition":"...","partOfSpeech":"NOUN","translations":["..."]}]' },
      { name: 'notes', type: 'string' },
      { name: 'createCard', type: 'boolean' },
      { name: 'topicId', type: 'uuid' },
    ],
  },
  {
    name: 'updateEntryNotes',
    type: 'mutation',
    description: 'Update notes on a dictionary entry.',
    query: `mutation UpdateEntryNotes($input: UpdateEntryNotesInput!) {
  updateEntryNotes(input: $input) { entry { id notes } }
}`,
    fields: [
      { name: 'entryId', type: 'uuid', required: true },
      { name: 'notes', type: 'string' },
    ],
  },
  {
    name: 'deleteEntry',
    type: 'mutation',
    description: 'Soft-delete a dictionary entry.',
    query: `mutation DeleteEntry($id: UUID!) {
  deleteEntry(id: $id) { entryId }
}`,
    fields: [
      { name: 'id', type: 'uuid', required: true },
    ],
  },
  {
    name: 'restoreEntry',
    type: 'mutation',
    description: 'Restore a soft-deleted entry from trash.',
    query: `mutation RestoreEntry($id: UUID!) {
  restoreEntry(id: $id) { entry { id text } }
}`,
    fields: [
      { name: 'id', type: 'uuid', required: true },
    ],
  },
  {
    name: 'batchDeleteEntries',
    type: 'mutation',
    description: 'Batch soft-delete multiple entries at once.',
    query: `mutation BatchDelete($ids: [UUID!]!) {
  batchDeleteEntries(ids: $ids) { deletedCount errors { id message } }
}`,
    fields: [
      { name: 'ids', type: 'uuid[]', required: true },
    ],
  },
  {
    name: 'importEntries',
    type: 'mutation',
    description: 'Import entries from structured data.',
    query: `mutation ImportEntries($input: ImportEntriesInput!) {
  importEntries(input: $input) { importedCount skippedCount errors { index text message } }
}`,
    fields: [
      { name: 'items', type: 'json', required: true, placeholder: '[{"text":"hello","translations":["привет"]}]' },
    ],
  },
]

// ============================================================
//  GROUP 3 — Content
// ============================================================

const contentOperations: Operation[] = [
  {
    name: 'addSense',
    type: 'mutation',
    description: 'Add a new sense to a dictionary entry.',
    query: `mutation AddSense($input: AddSenseInput!) {
  addSense(input: $input) { sense { id definition partOfSpeech cefrLevel position translations { id text } } }
}`,
    fields: [
      { name: 'entryId', type: 'uuid', required: true },
      { name: 'definition', type: 'string' },
      { name: 'partOfSpeech', type: 'enum', enumValues: PART_OF_SPEECH_VALUES },
      { name: 'cefrLevel', type: 'string', placeholder: 'A1, A2, B1, B2, C1, C2' },
      { name: 'translations', type: 'json', placeholder: '["word1","word2"]' },
    ],
  },
  {
    name: 'updateSense',
    type: 'mutation',
    description: 'Update an existing sense.',
    query: `mutation UpdateSense($input: UpdateSenseInput!) {
  updateSense(input: $input) { sense { id definition partOfSpeech cefrLevel position } }
}`,
    fields: [
      { name: 'senseId', type: 'uuid', required: true },
      { name: 'definition', type: 'string' },
      { name: 'partOfSpeech', type: 'enum', enumValues: PART_OF_SPEECH_VALUES },
      { name: 'cefrLevel', type: 'string', placeholder: 'A1, A2, B1, B2, C1, C2' },
    ],
  },
  {
    name: 'deleteSense',
    type: 'mutation',
    description: 'Delete a sense by ID.',
    query: `mutation DeleteSense($id: UUID!) {
  deleteSense(id: $id) { senseId }
}`,
    fields: [
      { name: 'id', type: 'uuid', required: true },
    ],
  },
  {
    name: 'reorderSenses',
    type: 'mutation',
    description: 'Reorder senses within an entry.',
    query: `mutation ReorderSenses($input: ReorderSensesInput!) {
  reorderSenses(input: $input) { success }
}`,
    fields: [
      { name: 'entryId', type: 'uuid', required: true },
      { name: 'items', type: 'json', required: true, placeholder: '[{"id":"uuid","position":0}]' },
    ],
  },
  {
    name: 'addTranslation',
    type: 'mutation',
    description: 'Add a translation to a sense.',
    query: `mutation AddTranslation($input: AddTranslationInput!) {
  addTranslation(input: $input) { translation { id text } }
}`,
    fields: [
      { name: 'senseId', type: 'uuid', required: true },
      { name: 'text', type: 'string', required: true },
    ],
  },
  {
    name: 'updateTranslation',
    type: 'mutation',
    description: 'Update an existing translation.',
    query: `mutation UpdateTranslation($input: UpdateTranslationInput!) {
  updateTranslation(input: $input) { translation { id text } }
}`,
    fields: [
      { name: 'translationId', type: 'uuid', required: true },
      { name: 'text', type: 'string', required: true },
    ],
  },
  {
    name: 'deleteTranslation',
    type: 'mutation',
    description: 'Delete a translation by ID.',
    query: `mutation DeleteTranslation($id: UUID!) {
  deleteTranslation(id: $id) { translationId }
}`,
    fields: [
      { name: 'id', type: 'uuid', required: true },
    ],
  },
  {
    name: 'reorderTranslations',
    type: 'mutation',
    description: 'Reorder translations within a sense.',
    query: `mutation ReorderTranslations($input: ReorderTranslationsInput!) {
  reorderTranslations(input: $input) { success }
}`,
    fields: [
      { name: 'senseId', type: 'uuid', required: true },
      { name: 'items', type: 'json', required: true, placeholder: '[{"id":"uuid","position":0}]' },
    ],
  },
  {
    name: 'addExample',
    type: 'mutation',
    description: 'Add an example sentence to a sense.',
    query: `mutation AddExample($input: AddExampleInput!) {
  addExample(input: $input) { example { id sentence translation } }
}`,
    fields: [
      { name: 'senseId', type: 'uuid', required: true },
      { name: 'sentence', type: 'string', required: true },
      { name: 'translation', type: 'string' },
    ],
  },
  {
    name: 'updateExample',
    type: 'mutation',
    description: 'Update an existing example.',
    query: `mutation UpdateExample($input: UpdateExampleInput!) {
  updateExample(input: $input) { example { id sentence translation } }
}`,
    fields: [
      { name: 'exampleId', type: 'uuid', required: true },
      { name: 'sentence', type: 'string' },
      { name: 'translation', type: 'string' },
    ],
  },
  {
    name: 'deleteExample',
    type: 'mutation',
    description: 'Delete an example by ID.',
    query: `mutation DeleteExample($id: UUID!) {
  deleteExample(id: $id) { exampleId }
}`,
    fields: [
      { name: 'id', type: 'uuid', required: true },
    ],
  },
  {
    name: 'reorderExamples',
    type: 'mutation',
    description: 'Reorder examples within a sense.',
    query: `mutation ReorderExamples($input: ReorderExamplesInput!) {
  reorderExamples(input: $input) { success }
}`,
    fields: [
      { name: 'senseId', type: 'uuid', required: true },
      { name: 'items', type: 'json', required: true, placeholder: '[{"id":"uuid","position":0}]' },
    ],
  },
  {
    name: 'addUserImage',
    type: 'mutation',
    description: 'Add a user image to a dictionary entry.',
    query: `mutation AddUserImage($input: AddUserImageInput!) {
  addUserImage(input: $input) { image { id url caption createdAt } }
}`,
    fields: [
      { name: 'entryId', type: 'uuid', required: true },
      { name: 'url', type: 'string', required: true, placeholder: 'https://example.com/image.png' },
      { name: 'caption', type: 'string' },
    ],
  },
  {
    name: 'deleteUserImage',
    type: 'mutation',
    description: 'Delete a user image by ID.',
    query: `mutation DeleteUserImage($id: UUID!) {
  deleteUserImage(id: $id) { imageId }
}`,
    fields: [
      { name: 'id', type: 'uuid', required: true },
    ],
  },
]

// ============================================================
//  GROUP 4 — Study
// ============================================================

const studyOperations: Operation[] = [
  {
    name: 'studyQueue',
    type: 'query',
    description: 'Get the study queue (entries due for review).',
    query: `query StudyQueue($limit: Int) {
  studyQueue(limit: $limit) {
    id text
    senses { id definition partOfSpeech translations { id text } }
    card { id status nextReviewAt intervalDays easeFactor }
  }
}`,
    fields: [
      { name: 'limit', type: 'number', placeholder: '10' },
    ],
  },
  {
    name: 'dashboard',
    type: 'query',
    description: 'Get study dashboard (due counts, streak, status breakdown, active session).',
    query: `query Dashboard {
  dashboard {
    dueCount newCount reviewedToday streak overdueCount
    statusCounts { new learning review mastered }
    activeSession { id status startedAt finishedAt
      result { totalReviews gradeCounts { again hard good easy } averageDurationMs } }
  }
}`,
    fields: [],
  },
  {
    name: 'cardHistory',
    type: 'query',
    description: 'Get review history for a card.',
    query: `query CardHistory($input: GetCardHistoryInput!) {
  cardHistory(input: $input) { id cardId grade durationMs reviewedAt }
}`,
    fields: [
      { name: 'cardId', type: 'uuid', required: true },
      { name: 'limit', type: 'number', placeholder: '20' },
      { name: 'offset', type: 'number', placeholder: '0' },
    ],
  },
  {
    name: 'cardStats',
    type: 'query',
    description: 'Get card statistics (accuracy, grade distribution).',
    query: `query CardStats($cardId: UUID!) {
  cardStats(cardId: $cardId) {
    totalReviews averageDurationMs accuracy
    gradeDistribution { again hard good easy }
  }
}`,
    fields: [
      { name: 'cardId', type: 'uuid', required: true },
    ],
  },
  {
    name: 'reviewCard',
    type: 'mutation',
    description: 'Submit a review grade for a card.',
    query: `mutation ReviewCard($input: ReviewCardInput!) {
  reviewCard(input: $input) { card { id status nextReviewAt intervalDays easeFactor } }
}`,
    fields: [
      { name: 'cardId', type: 'uuid', required: true },
      { name: 'grade', type: 'enum', required: true, enumValues: ['AGAIN', 'HARD', 'GOOD', 'EASY'] },
      { name: 'durationMs', type: 'number' },
    ],
  },
  {
    name: 'undoReview',
    type: 'mutation',
    description: 'Undo the last review for a card.',
    query: `mutation UndoReview($cardId: UUID!) {
  undoReview(cardId: $cardId) { card { id status nextReviewAt intervalDays easeFactor } }
}`,
    fields: [
      { name: 'cardId', type: 'uuid', required: true },
    ],
  },
  {
    name: 'createCard',
    type: 'mutation',
    description: 'Create a study card for a dictionary entry.',
    query: `mutation CreateCard($entryId: UUID!) {
  createCard(entryId: $entryId) { card { id entryId status nextReviewAt intervalDays easeFactor createdAt } }
}`,
    fields: [
      { name: 'entryId', type: 'uuid', required: true },
    ],
  },
  {
    name: 'deleteCard',
    type: 'mutation',
    description: 'Delete a study card.',
    query: `mutation DeleteCard($id: UUID!) {
  deleteCard(id: $id) { cardId }
}`,
    fields: [
      { name: 'id', type: 'uuid', required: true },
    ],
  },
  {
    name: 'batchCreateCards',
    type: 'mutation',
    description: 'Create study cards for multiple entries at once.',
    query: `mutation BatchCreateCards($entryIds: [UUID!]!) {
  batchCreateCards(entryIds: $entryIds) { createdCount skippedCount errors { entryId message } }
}`,
    fields: [
      { name: 'entryIds', type: 'uuid[]', required: true },
    ],
  },
  {
    name: 'startStudySession',
    type: 'mutation',
    description: 'Start a new study session.',
    query: `mutation StartSession {
  startStudySession { session { id status startedAt } }
}`,
    fields: [],
  },
  {
    name: 'finishStudySession',
    type: 'mutation',
    description: 'Finish an active study session.',
    query: `mutation FinishSession($input: FinishSessionInput!) {
  finishStudySession(input: $input) {
    session { id status finishedAt
      result { totalReviews gradeCounts { again hard good easy } averageDurationMs } }
  }
}`,
    fields: [
      { name: 'sessionId', type: 'uuid', required: true },
    ],
  },
  {
    name: 'abandonStudySession',
    type: 'mutation',
    description: 'Abandon the current active study session.',
    query: `mutation AbandonSession {
  abandonStudySession { success }
}`,
    fields: [],
  },
]

// ============================================================
//  GROUP 5 — Organization
// ============================================================

const organizationOperations: Operation[] = [
  {
    name: 'topics',
    type: 'query',
    description: 'List all user topics (sorted by name, with entry count).',
    query: `query Topics {
  topics { id name description entryCount createdAt updatedAt }
}`,
    fields: [],
  },
  {
    name: 'inboxItems',
    type: 'query',
    description: 'List inbox items with offset pagination.',
    query: `query InboxItems($limit: Int, $offset: Int) {
  inboxItems(limit: $limit, offset: $offset) {
    items { id text context createdAt } totalCount
  }
}`,
    fields: [
      { name: 'limit', type: 'number', placeholder: '20' },
      { name: 'offset', type: 'number', placeholder: '0' },
    ],
  },
  {
    name: 'inboxItem',
    type: 'query',
    description: 'Get a single inbox item by ID.',
    query: `query InboxItem($id: UUID!) {
  inboxItem(id: $id) { id text context createdAt }
}`,
    fields: [
      { name: 'id', type: 'uuid', required: true },
    ],
  },
  {
    name: 'createTopic',
    type: 'mutation',
    description: 'Create a new topic.',
    query: `mutation CreateTopic($input: CreateTopicInput!) {
  createTopic(input: $input) { topic { id name description entryCount createdAt } }
}`,
    fields: [
      { name: 'name', type: 'string', required: true },
      { name: 'description', type: 'string' },
    ],
  },
  {
    name: 'updateTopic',
    type: 'mutation',
    description: 'Update a topic name and/or description.',
    query: `mutation UpdateTopic($input: UpdateTopicInput!) {
  updateTopic(input: $input) { topic { id name description updatedAt } }
}`,
    fields: [
      { name: 'topicId', type: 'uuid', required: true },
      { name: 'name', type: 'string' },
      { name: 'description', type: 'string' },
    ],
  },
  {
    name: 'deleteTopic',
    type: 'mutation',
    description: 'Delete a topic by ID.',
    query: `mutation DeleteTopic($id: UUID!) {
  deleteTopic(id: $id) { topicId }
}`,
    fields: [
      { name: 'id', type: 'uuid', required: true },
    ],
  },
  {
    name: 'linkEntryToTopic',
    type: 'mutation',
    description: 'Link a dictionary entry to a topic.',
    query: `mutation LinkEntryToTopic($input: LinkEntryInput!) {
  linkEntryToTopic(input: $input) { success }
}`,
    fields: [
      { name: 'topicId', type: 'uuid', required: true },
      { name: 'entryId', type: 'uuid', required: true },
    ],
  },
  {
    name: 'unlinkEntryFromTopic',
    type: 'mutation',
    description: 'Unlink a dictionary entry from a topic.',
    query: `mutation UnlinkEntryFromTopic($input: UnlinkEntryInput!) {
  unlinkEntryFromTopic(input: $input) { success }
}`,
    fields: [
      { name: 'topicId', type: 'uuid', required: true },
      { name: 'entryId', type: 'uuid', required: true },
    ],
  },
  {
    name: 'batchLinkEntriesToTopic',
    type: 'mutation',
    description: 'Link multiple entries to a topic at once.',
    query: `mutation BatchLinkEntriesToTopic($input: BatchLinkEntriesInput!) {
  batchLinkEntriesToTopic(input: $input) { linked skipped }
}`,
    fields: [
      { name: 'topicId', type: 'uuid', required: true },
      { name: 'entryIds', type: 'uuid[]', required: true },
    ],
  },
  {
    name: 'createInboxItem',
    type: 'mutation',
    description: 'Add a new item to the inbox.',
    query: `mutation CreateInboxItem($input: CreateInboxItemInput!) {
  createInboxItem(input: $input) { item { id text context createdAt } }
}`,
    fields: [
      { name: 'text', type: 'string', required: true },
      { name: 'context', type: 'string' },
    ],
  },
  {
    name: 'deleteInboxItem',
    type: 'mutation',
    description: 'Delete an inbox item by ID.',
    query: `mutation DeleteInboxItem($id: UUID!) {
  deleteInboxItem(id: $id) { itemId }
}`,
    fields: [
      { name: 'id', type: 'uuid', required: true },
    ],
  },
  {
    name: 'clearInbox',
    type: 'mutation',
    description: 'Delete all inbox items.',
    query: `mutation ClearInbox {
  clearInbox { deletedCount }
}`,
    fields: [],
  },
]

// ============================================================
//  GROUP 6 — User
// ============================================================

const userOperations: Operation[] = [
  {
    name: 'me',
    type: 'query',
    description: 'Get current user profile and settings.',
    query: `query Me {
  me {
    id email username name avatarUrl createdAt
    settings { newCardsPerDay reviewsPerDay maxIntervalDays timezone }
  }
}`,
    fields: [],
  },
  {
    name: 'updateSettings',
    type: 'mutation',
    description: 'Update user study settings.',
    query: `mutation UpdateSettings($input: UpdateSettingsInput!) {
  updateSettings(input: $input) {
    settings { newCardsPerDay reviewsPerDay maxIntervalDays timezone }
  }
}`,
    fields: [
      { name: 'newCardsPerDay', type: 'number' },
      { name: 'reviewsPerDay', type: 'number' },
      { name: 'maxIntervalDays', type: 'number' },
      { name: 'timezone', type: 'string', placeholder: 'Europe/Moscow' },
    ],
  },
]

// ============================================================
//  All groups
// ============================================================

export const operationGroups: OperationGroup[] = [
  { name: 'RefCatalog', operations: refCatalogOperations },
  { name: 'Dictionary', operations: dictionaryOperations },
  { name: 'Content', operations: contentOperations },
  { name: 'Study', operations: studyOperations },
  { name: 'Organization', operations: organizationOperations },
  { name: 'User', operations: userOperations },
]
