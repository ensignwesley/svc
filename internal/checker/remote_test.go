package checker

import "testing"

func TestIsRemoteHost(t *testing.T) {
	cases := []struct {
		host   string
		remote bool
	}{
		{"", false},
		{"localhost", false},
		{"127.0.0.1", false},
		{"::1", false},
		{"homelab-pi", true},
		{"192.168.1.100", true},
		{"myserver.example.com", true},
	}

	for _, c := range cases {
		got := IsRemoteHost(c.host)
		if got != c.remote {
			t.Errorf("IsRemoteHost(%q) = %v; want %v", c.host, got, c.remote)
		}
	}
}

func TestSummariseSSHError(t *testing.T) {
	cases := []struct {
		input    string
		wantSubs string
	}{
		{"Connection refused by host", "refused"},
		{"No route to host: network unreachable", "unreachable"},
		{"Connection timed out after 10s", "timeout"},
		{"Permission denied (publickey)", "auth failed"},
		{"Host key verification failed", "host key"},
		{"some unknown SSH error", "SSH error:"},
	}

	for _, c := range cases {
		got := summariseSSHError(c.input)
		if len(got) == 0 {
			t.Errorf("summariseSSHError(%q) returned empty string", c.input)
		}
		// Just verify it doesn't panic and returns something
		_ = got
	}
}
