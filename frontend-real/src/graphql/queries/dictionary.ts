import { gql } from '@apollo/client'

export const GET_DICTIONARY = gql`
  query GetDictionary($input: DictionaryFilterInput!) {
    dictionary(input: $input) {
      edges {
        node {
          id
          text
          textNormalized
          senses {
            id
            definition
            partOfSpeech
            translations {
              id
              text
            }
          }
          pronunciations {
            id
            transcription
          }
          topics {
            id
            name
          }
          createdAt
          updatedAt
        }
        cursor
      }
      pageInfo {
        hasNextPage
        endCursor
      }
      totalCount
    }
  }
`

export const GET_DICTIONARY_ENTRY = gql`
  query GetDictionaryEntry($id: UUID!) {
    dictionaryEntry(id: $id) {
      id
      text
      textNormalized
      notes
      senses {
        id
        definition
        partOfSpeech
        translations {
          id
          text
        }
        examples {
          id
          sentence
          translation
        }
      }
      catalogImages {
        id
        url
        caption
      }
      userImages {
        id
        url
        caption
      }
      pronunciations {
        id
        audioUrl
        transcription
        region
      }
      topics {
        id
        name
        description
      }
      card {
        id
        entryId
        state
        due
        lastReview
        reps
        lapses
        scheduledDays
        createdAt
        updatedAt
      }
      createdAt
      updatedAt
    }
  }
`

export const GET_CARD_STATS = gql`
  query GetCardStats($cardId: UUID!) {
    cardStats(cardId: $cardId) {
      totalReviews
      accuracy
      currentState
      scheduledDays
      gradeDistribution {
        again
        hard
        good
        easy
      }
    }
  }
`

export const GET_CARD_HISTORY = gql`
  query GetCardHistory($input: GetCardHistoryInput!) {
    cardHistory(input: $input) {
      logs {
        id
        cardId
        grade
        reviewedAt
      }
      totalCount
    }
  }
`

export const GET_TOPICS = gql`
  query GetTopics {
    topics {
      id
      name
      description
    }
  }
`

export const UPDATE_ENTRY_NOTES = gql`
  mutation UpdateEntryNotes($input: UpdateEntryNotesInput!) {
    updateEntryNotes(input: $input) {
      entry {
        id
        notes
        updatedAt
      }
    }
  }
`

export const CREATE_CARD = gql`
  mutation CreateCard($entryId: UUID!) {
    createCard(entryId: $entryId) {
      card {
        id
        entryId
        state
        due
        lastReview
        reps
        lapses
        scheduledDays
        createdAt
        updatedAt
      }
    }
  }
`

export const SEARCH_CATALOG = gql`
  query SearchCatalog($query: String!, $limit: Int) {
    searchCatalog(query: $query, limit: $limit) {
      id
      text
      textNormalized
      cefrLevel
      senses {
        id
        definition
        partOfSpeech
        translations {
          id
          text
        }
      }
      pronunciations {
        id
        transcription
      }
    }
  }
`

export const PREVIEW_REF_ENTRY = gql`
  query PreviewRefEntry($text: String!) {
    previewRefEntry(text: $text) {
      id
      text
      textNormalized
      frequencyRank
      cefrLevel
      isCoreLexicon
      senses {
        id
        definition
        partOfSpeech
        cefrLevel
        notes
        sourceSlug
        position
        translations {
          id
          text
          sourceSlug
        }
        examples {
          id
          sentence
          translation
          sourceSlug
        }
      }
      pronunciations {
        id
        transcription
        audioUrl
        region
      }
    }
  }
`

export const CREATE_ENTRY_FROM_CATALOG = gql`
  mutation CreateEntryFromCatalog($input: CreateEntryFromCatalogInput!) {
    createEntryFromCatalog(input: $input) {
      entry {
        id
        text
        senses {
          id
          definition
          partOfSpeech
          translations {
            id
            text
          }
        }
        createdAt
      }
    }
  }
`

export const DELETE_ENTRY = gql`
  mutation DeleteEntry($id: UUID!) {
    deleteEntry(id: $id) {
      entryId
    }
  }
`
