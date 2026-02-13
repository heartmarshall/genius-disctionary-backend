package app

import "fmt"

// Version, Commit, and BuildTime are set via ldflags at build time.
// Example: go build -ldflags "-X github.com/heartmarshall/myenglish-backend/internal/app.Version=1.0.0"
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

// BuildVersion returns a formatted version string for startup logs and health endpoints.
func BuildVersion() string {
	return fmt.Sprintf("%s (commit: %s, built: %s)", Version, Commit, BuildTime)
}
