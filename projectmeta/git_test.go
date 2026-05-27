package projectmeta

import "testing"

func TestSanitizeRemoteURLStripsHTTPUserInfo(t *testing.T) {
	remote := SanitizeRemote("origin", "https://user:token@github.com/yangyifan18/dotvibe.git")
	if remote.URL != "https://github.com/yangyifan18/dotvibe.git" || !remote.CredentialsRedacted || !remote.Cloneable || !remote.Sanitized {
		t.Fatalf("remote = %#v", remote)
	}
}

func TestSanitizeRemoteURLPreservesSSHRemote(t *testing.T) {
	remote := SanitizeRemote("origin", "git@github.com:yangyifan18/dotvibe.git")
	if remote.URL != "git@github.com:yangyifan18/dotvibe.git" || remote.CredentialsRedacted || !remote.Cloneable || !remote.Sanitized {
		t.Fatalf("remote = %#v", remote)
	}
}

func TestSanitizeRemoteURLMarksLocalPathsUncloneable(t *testing.T) {
	remote := SanitizeRemote("origin", "/Users/young/repos/dotvibe")
	if remote.Cloneable || remote.Reason != "local-path" || !remote.Sanitized {
		t.Fatalf("remote = %#v", remote)
	}
}
