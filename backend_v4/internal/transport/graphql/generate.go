// Package graphql provides the GraphQL transport layer for the MyEnglish backend.
// It defines GraphQL schema, resolvers, and error handling for the spaced repetition
// vocabulary learning application. Scalar types (UUID, DateTime) and GraphQL types
// are automatically generated via gqlgen from the schema file.
package graphql

//go:generate go run github.com/99designs/gqlgen generate
