package projectmeta

import (
	"net/url"
	"strings"

	"github.com/yangyifan18/dotvibe/backup"
)

func SanitizeRemote(name, raw string) backup.ProjectGitRemote {
	trimmed := strings.TrimSpace(raw)
	remote := backup.ProjectGitRemote{Name: name, URL: trimmed, Sanitized: true, Cloneable: false}
	if trimmed == "" {
		remote.Reason = "empty"
		return remote
	}
	if isSCPStyleRemote(trimmed) {
		remote.Cloneable = true
		return remote
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" {
		remote.Reason = "local-path"
		return remote
	}
	if parsed.User != nil {
		parsed.User = nil
		remote.CredentialsRedacted = true
	}
	switch parsed.Scheme {
	case "https", "http", "ssh", "git":
		remote.URL = parsed.String()
		remote.Cloneable = true
	default:
		remote.URL = parsed.String()
		remote.Reason = "unsupported-scheme"
	}
	return remote
}

func isSCPStyleRemote(raw string) bool {
	if strings.Contains(raw, "://") || strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "./") || strings.HasPrefix(raw, "../") {
		return false
	}
	at := strings.Index(raw, "@")
	colon := strings.Index(raw, ":")
	return at > 0 && colon > at+1 && strings.Count(raw[:colon], ":") == 0
}
