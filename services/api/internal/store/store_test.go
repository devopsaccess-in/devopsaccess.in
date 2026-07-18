package store

import "testing"

func TestSlugify(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "vikram", "vikram"},
		{"uppercase", "Vikram", "vikram"},
		{"email localpart with dot", "vikram.singh", "vikram-singh"},
		{"plus tag", "vikram+test", "vikram-test"},
		{"spaces collapse", "acme  corp", "acme-corp"},
		{"leading trailing junk", "--acme--", "acme"},
		{"unicode dropped", "अस्पताल", "team"},
		{"empty", "", "team"},
		{"only symbols", "@#$%", "team"},
		{"truncated to 30", "abcdefghijklmnopqrstuvwxyz0123456789", "abcdefghijklmnopqrstuvwxyz0123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Slugify(tt.in); got != tt.want {
				t.Fatalf("Slugify(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestTenantBaseName(t *testing.T) {
	tests := []struct {
		name  string
		email string
		user  string
		want  string
	}{
		{"email localpart wins", "ops@acme.in", "Vikram", "ops"},
		{"name fallback", "", "Vikram", "Vikram"},
		{"generic fallback", "", "", "team"},
		{"bare at-sign ignored", "@acme.in", "Vikram", "Vikram"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tenantBaseName(tt.email, tt.user); got != tt.want {
				t.Fatalf("tenantBaseName(%q, %q) = %q, want %q", tt.email, tt.user, got, tt.want)
			}
		})
	}
}
