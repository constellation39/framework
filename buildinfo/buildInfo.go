// Package buildinfo provides build-time information about the application.
// To set build information during compilation, use the following ldflags:
//
// go build -ldflags "-X github.com/your/pkg/buildinfo.version=1.0.0
//
//	-X github.com/your/pkg/buildinfo.buildTime=$(date -u '+%Y-%m-%d %H:%M:%S')
//	-X github.com/your/pkg/buildinfo.gitUser=$(git config user.name)
//	-X github.com/your/pkg/buildinfo.gitBranch=$(git rev-parse --abbrev-ref HEAD)
//	-X github.com/your/pkg/buildinfo.gitCommit=$(git rev-parse HEAD)"
package buildinfo

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// Build information variables injected at build time
var (
	version   = "dev"     // Semantic version
	buildTime = "unknown" // Build timestamp
	gitUser   = "unknown" // Git user
	gitBranch = "unknown" // Git branch
	gitCommit = "unknown" // Git commit hash
)

// Info represents build-time information about the application
type Info struct {
	Version     string    `json:"version"`      // Semantic version number
	BuildTime   time.Time `json:"build_time"`   // Build timestamp
	GitUser     string    `json:"git_user"`     // Git user who built the binary
	GitBranch   string    `json:"git_branch"`   // Git branch used for the build
	GitCommit   string    `json:"git_commit"`   // Git commit hash
	BuildString string    `json:"build_string"` // Human-readable build string
}

var (
	instance *Info
	once     sync.Once
)

// Get returns the build information singleton
func Get() (*Info, error) {
	once.Do(func() {
		// Parse build time
		t, err := time.Parse("2006-01-02 15:04:05", buildTime)
		if err != nil {
			log.Printf("Failed to parse build time: %v", err)
			t = time.Time{} // Zero time if parsing fails
		}

		instance = &Info{
			Version:   version,
			BuildTime: t,
			GitUser:   gitUser,
			GitBranch: gitBranch,
			GitCommit: gitCommit,
		}

		// Create human-readable build string
		instance.BuildString = generateBuildString(instance)
	})

	if instance == nil {
		return nil, fmt.Errorf("build info instance is nil")
	}
	return instance, nil
}

// generateBuildString creates a human-readable build string
func generateBuildString(i *Info) string {
	return fmt.Sprintf("Version: %s, Built: %s, By: %s, Branch: %s, Commit: %.8s",
		i.Version,
		i.BuildTime.Format(time.RFC3339),
		i.GitUser,
		i.GitBranch,
		i.GitCommit,
	)
}

// String returns a human-readable representation of build information
func (i *Info) String() string {
	return i.BuildString
}

// JSON returns the build information as a JSON string
func (i *Info) JSON() (string, error) {
	data, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal build info: %w", err)
	}
	return string(data), nil
}

// MustJSON is like JSON but panics on error
func (i *Info) MustJSON() string {
	data, err := i.JSON()
	if err != nil {
		panic(err)
	}
	return data
}
