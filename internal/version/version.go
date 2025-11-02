package version

import (
	"fmt"
	"runtime"
)

// 版本信息变量，这些将在构建时通过 -ldflags 注入
var (
	Version   = "dev"        // 版本号
	BuildTime = "unknown"    // 构建时间
	GitCommit = "unknown"    // Git 提交哈希
	GoVersion = runtime.Version() // Go 版本
)

// VersionInfo 返回完整的版本信息
type VersionInfo struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
	GitCommit string `json:"git_commit"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// GetVersionInfo 获取完整的版本信息
func GetVersionInfo() VersionInfo {
	return VersionInfo{
		Version:   Version,
		BuildTime: BuildTime,
		GitCommit: GitCommit,
		GoVersion: GoVersion,
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}
}

// String 返回版本信息的字符串表示
func (v VersionInfo) String() string {
	return fmt.Sprintf("StreamASR %s (built: %s, commit: %s, go: %s, platform: %s)",
		v.Version, v.BuildTime, v.GitCommit[:min(7, len(v.GitCommit))], v.GoVersion, v.Platform)
}

// Short 返回简短的版本信息
func Short() string {
	commit := GitCommit
	if len(commit) > 7 {
		commit = commit[:7]
	}
	return fmt.Sprintf("%s-%s", Version, commit)
}

// Full 返回完整的版本信息
func Full() string {
	info := GetVersionInfo()
	return info.String()
}

// GetBuildTime 获取构建时间
func GetBuildTime() string {
	return BuildTime
}

// GetGitCommit 获取 Git 提交哈希
func GetGitCommit() string {
	return GitCommit
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}