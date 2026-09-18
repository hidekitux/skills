package redact

import "testing"

func TestIsPrivateHostAnswersEveryAddressClass(t *testing.T) {
	private := map[string]string{
		"127.0.0.1":   "IPv4 loopback",
		"::1":         "IPv6 loopback",
		"10.0.0.1":    "IPv4 private range",
		"192.168.1.5": "IPv4 private range",
		"172.16.0.1":  "IPv4 private range",
		"fd00::1":     "unique local IPv6 address",
		"fc00::abcd":  "unique local IPv6 address",
		"169.254.1.1": "IPv4 link local address",
		"fe80::1":     "IPv6 link local address",
		"localhost":   "local name",
		"build.local": "local name suffix",
		"BUILD.LOCAL": "local name suffix in upper case",
	}
	for host, class := range private {
		if !IsPrivateHost(host) {
			t.Fatalf("expected %q (%s) to be private", host, class)
		}
	}
	public := map[string]string{
		"github.com":  "public name",
		"8.8.8.8":     "public IPv4 address",
		"2001:db8::1": "documentation IPv6 address",
		"example.com": "public name",
	}
	for host, class := range public {
		if IsPrivateHost(host) {
			t.Fatalf("expected %q (%s) to be public", host, class)
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

func TestURLPatternIgnoresTheCaseOfTheScheme(t *testing.T) {
	for _, value := range []string{"HTTPS://github.com/a", "Http://example.org/b"} {
		if URLPattern().FindString(value) != value {
			t.Fatalf("expected %q to match whatever the case of its scheme", value)
		}
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
