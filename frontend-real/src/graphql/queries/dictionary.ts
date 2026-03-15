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
      createdAt
      updatedAt
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

export const DELETE_ENTRY = gql`
  mutation DeleteEntry($id: UUID!) {
    deleteEntry(id: $id) {
      entryId
    }
  }
`
