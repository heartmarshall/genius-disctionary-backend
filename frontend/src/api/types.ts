export interface GraphQLError {
  message: string
  path?: string[]
  extensions?: {
    code?: string
    fields?: { field: string; message: string }[]
  }
}

export interface GraphQLResponse<T> {
  data: T | null
  errors: GraphQLError[] | null
}

export interface GraphQLResult<T> {
  data: T | null
  errors: GraphQLError[] | null
  raw: {
    query: string
    variables: unknown
    response: unknown
    status: number
  }
}
