package seeder_test

import (
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/refentry"
	"github.com/heartmarshall/myenglish-backend/internal/app/seeder"
)

// Compile-time check: *refentry.Repo must satisfy RefEntryBulkRepo.
var _ seeder.RefEntryBulkRepo = (*refentry.Repo)(nil)
