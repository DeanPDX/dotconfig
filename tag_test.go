package dotconfig

import "testing"

func TestParseTag(t *testing.T) {
	name, _ := parseTag("MAX_BYTES_PER_REQUEST,required,optional")
	if name != "MAX_BYTES_PER_REQUEST" {
		t.Fatalf("name = %q, want MAX_BYTES_PER_REQUEST", name)
	}
	for _, tt := range []struct {
		tag  string
		opt  string
		want bool
	}{
		{"MAX_BYTES_PER_REQUEST, required, optional", "required", true},
		{"MAX_BYTES_PER_REQUEST", "required", false},
		{"MAX_BYTES_PER_REQUEST,required,optional", "optional", true},
		{"MAX_BYTES_PER_REQUEST,required,optional", "bogus", false},
		{"MAX_BYTES_PER_REQUEST, required", "", false},
	} {
		_, opts := parseTag(tt.tag)
		if opts.Contains(tt.opt) != tt.want {
			t.Errorf("Contains(%q) = %v, want %v", tt.opt, !tt.want, tt.want)
		}
	}
}
