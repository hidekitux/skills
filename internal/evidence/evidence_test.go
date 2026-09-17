package evidence

import "testing"

func TestIsPrivateAddressRejectsLocalAndPrivateHosts(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "10.0.0.1", "192.168.1.5", "::1", "localhost", "build.local"} {
		if !IsPrivateAddress(host) {
			t.Fatalf("expected %q to be a private address", host)
		}
	}
	for _, host := range []string{"github.com", "8.8.8.8", "example.com"} {
		if IsPrivateAddress(host) {
			t.Fatalf("expected %q to be a public address", host)
		}
	}
}

func TestCredentialPatternMatchesEveryRejectedCredentialForm(t *testing.T) {
	for _, value := range []string{
		"Authorization: Bearer abc123",
		"password=hunter2",
		"token = abc",
		"api_key=abc",
		"ghp_0123456789abcdef",
		"github_pat_0123456789",
		"sk-0123456789",
		"AKIAABCDEFGHIJKLMNOP",
	} {
		if !CredentialPattern().MatchString(value) {
			t.Fatalf("expected %q to match the credential pattern", value)
		}
	}
	if CredentialPattern().MatchString("plan-issue posted the plan comment") {
		t.Fatal("expected ordinary prose not to match the credential pattern")
	}
}

func TestURLPatternFindsEveryAbsoluteURL(t *testing.T) {
	found := URLPattern().FindAllString(`see https://github.com/a and http://example.org/b`, -1)
	if len(found) != 2 {
		t.Fatalf("expected 2 URLs, got %d: %v", len(found), found)
	}
	if found[0] != "https://github.com/a" || found[1] != "http://example.org/b" {
		t.Fatalf("unexpected URLs: %v", found)
	}
}

func TestCommitSHAPatternAcceptsOnlyAFullIdentifier(t *testing.T) {
	if !CommitSHAPattern().MatchString("aec2925cdba690384feaa7830d79fd6b18e22c77") {
		t.Fatal("expected a 40-character identifier to match")
	}
	for _, value := range []string{"aec2925", "AEC2925CDBA690384FEAA7830D79FD6B18E22C77", ""} {
		if CommitSHAPattern().MatchString(value) {
			t.Fatalf("expected %q not to match", value)
		}
	}
}
