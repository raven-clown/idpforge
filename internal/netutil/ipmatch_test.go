package netutil

import "testing"

func TestIPAllowed(t *testing.T) {
	cases := []struct {
		name    string
		ip      string
		allowed []string
		want    bool
	}{
		{"empty allowlist permits everything", "203.0.113.7", nil, true},
		{"exact match", "10.0.0.5", []string{"10.0.0.5"}, true},
		{"exact mismatch", "10.0.0.6", []string{"10.0.0.5"}, false},
		{"cidr match", "10.0.0.42", []string{"10.0.0.0/24"}, true},
		{"cidr mismatch", "10.0.1.42", []string{"10.0.0.0/24"}, false},
		{"invalid client ip rejected even with entries", "not-an-ip", []string{"10.0.0.0/24"}, false},
		{"mixed list, second entry matches", "192.168.1.1", []string{"10.0.0.0/24", "192.168.1.0/24"}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IPAllowed(tc.ip, tc.allowed); got != tc.want {
				t.Errorf("IPAllowed(%q, %v) = %v, want %v", tc.ip, tc.allowed, got, tc.want)
			}
		})
	}
}
