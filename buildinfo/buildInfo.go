// Package buildinfo provides build-time information about the application.
// To set build information during compilation, use the following ldflags:
//
// go build -ldflags "-X github.com/your/pkg/buildinfo.version=1.0.0
//
//	-X github.com/your/pkg/buildinfo.buildTime=$(date -u '+%Y-%m-%d %H:%M:%S')
//	-X github.com/your/pkg/buildinfo.gitUser=$(git config user.name)
//	-X github.com/your/pkg/buildinfo.gitBranch=$(git rev-parse --abbrev-ref HEAD)
//	-X github.com/your/pkg/buildinfo.gitCommit=$(git rev-parse HEAD)
//	-X github.com/your/pkg/buildinfo.debug=true"
package buildinfo

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// Build information variables injected at build time
var (
	version   = "dev"     // Semantic version
	buildTime = "unknown" // Build timestamp
	gitUser   = "unknown" // Git user
	gitBranch = "unknown" // Git branch
	gitCommit = "unknown" // Git commit hash
	debug     = "true"    // Debug mode flag (true/false)
)

// Info represents build-time information about the application
type Info struct {
	Version     string    `json:"version"`      // Semantic version number
	BuildTime   time.Time `json:"build_time"`   // Build timestamp
	GitUser     string    `json:"git_user"`     // Git user who built the binary
	GitBranch   string    `json:"git_branch"`   // Git branch used for the build
	GitCommit   string    `json:"git_commit"`   // Git commit hash
	BuildString string    `json:"build_string"` // Human-readable build string
	Debug       bool      `json:"debug"`        // Whether this is a debug build
}

// 在包初始化时立即创建单例实例
var instance = func() *Info {
	// Parse build time
	t, err := time.Parse("2006-01-02 15:04:05", buildTime)
	if err != nil {
		t = time.Time{} // Zero time if parsing fails
	}

	// Parse debug flag
	isDebug, err := strconv.ParseBool(debug)
	if err != nil {
		isDebug = true // Default to debug mode if parsing fails
	}

	info := &Info{
		Version:   version,
		BuildTime: t,
		GitUser:   gitUser,
		GitBranch: gitBranch,
		GitCommit: gitCommit,
		Debug:     isDebug,
	}

	// 构建字符串根据是否为调试版本而有不同
	buildMode := "Debug"
	if !isDebug {
		buildMode = "Release"
	}

	info.BuildString = fmt.Sprintf("Version: %s (%s), Built: %s, By: %s, Branch: %s, Commit: %.8s",
		info.Version,
		buildMode,
		info.BuildTime.Format(time.RFC3339),
		info.GitUser,
		info.GitBranch,
		info.GitCommit,
	)

	return info
}()

// Get returns the build information
func Get() *Info {
	return instance
}

// IsDebug returns whether this is a debug build
func IsDebug() bool {
	return instance.Debug
}

// IsRelease returns whether this is a release build
func IsRelease() bool {
	return !instance.Debug
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

// LogString returns a detailed or simplified string based on build mode
func (i *Info) LogString() string {
	if i.Debug {
		// 调试版返回详细信息
		return fmt.Sprintf(
			"[DEBUG BUILD] Version: %s\nBuild Time: %s\nGit User: %s\nBranch: %s\nCommit: %s",
			i.Version,
			i.BuildTime.Format(time.RFC3339),
			i.GitUser,
			i.GitBranch,
			i.GitCommit,
		)
	}

	// 正式版返回简化信息
	return fmt.Sprintf("Version %s (Release)", i.Version)
}
