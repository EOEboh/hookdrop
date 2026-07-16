package cmd

import "testing"

func TestNormalizeForward(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		path    string
		want    string
		wantErr bool
	}{
		{"empty is watch-only", "", "", "", false},
		{"bare port", "3000", "", "http://localhost:3000", false},
		{"host:port", "localhost:3000", "", "http://localhost:3000", false},
		{"host:port/path", "localhost:3000/webhook", "", "http://localhost:3000/webhook", false},
		{"full http url passes through", "http://127.0.0.1:8080/hook", "", "http://127.0.0.1:8080/hook", false},
		{"full https url passes through", "https://host:8443/hook", "", "https://host:8443/hook", false},
		{"port + --path", "3000", "/webhook", "http://localhost:3000/webhook", false},
		{"host:port/base + --path appends", "localhost:3000/api", "/stripe", "http://localhost:3000/api/stripe", false},
		{"--path without forward errors", "", "/webhook", "", true},
		{"garbage errors", "http://", "", "", true},
		{"unsupported scheme errors", "ftp://host/x", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeForward(tt.raw, tt.path)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
