import { gql } from '@apollo/client'

export const GET_DICTIONARY = gql`
  query GetDictionary($filter: WordFilter) {
    dictionary(filter: $filter) {
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
        examples {
          id
          sentence
          translation
        }
      }
      images {
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
      }
      createdAt
      updatedAt
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
      images {
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

export const UPDATE_WORD = gql`
  mutation UpdateWord($id: UUID!, $input: UpdateWordInput!) {
    updateWord(id: $id, input: $input) {
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
      images {
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
      updatedAt
    }
  }
`

export const DELETE_WORD = gql`
  mutation DeleteWord($id: UUID!) {
    deleteWord(id: $id)
  }
`
