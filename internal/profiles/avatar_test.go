package profiles

import "testing"

func TestNormalizeAvatarAcceptsPaletteAndHTMLColors(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{"violet", "violet", true},
		{" #4f46e5 ", "#4F46E5", true},
		{"#FFFFFF", "#FFFFFF", true},
		{"", "violet", true},
		{"#fff", "", false},
		{"#GG46E5", "", false},
		{"red", "", false},
	}
	for _, tt := range tests {
		got, err := NormalizeAvatar(tt.input)
		if tt.ok && (err != nil || got != tt.want) {
			t.Fatalf("NormalizeAvatar(%q) = %q, %v; want %q", tt.input, got, err, tt.want)
		}
		if !tt.ok && err == nil {
			t.Fatalf("NormalizeAvatar(%q) unexpectedly succeeded with %q", tt.input, got)
		}
	}
}
