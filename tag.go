package dotconfig

import "strings"

// tagOptions is the string following a comma in a struct field's "env"
// tag, or an empty string.
// Borrows HEAVILY from:
// https://cs.opensource.google/go/go/+/master:src/encoding/json/tags.go;bpv=0;bpt=1
// Main difference is I'm trimming the options so this is valid:
//
//	type myStruct struct {
//	  MaxBytesPerRequest int `env:"MAX_BYTES_PER_REQUEST, optionHasASpaceAfter"`
//	}
type tagOptions string

// parseTag splits a struct field's json tag into its name and
// comma-separated options.
func parseTag(tag string) (string, tagOptions) {
	tag, opt, _ := strings.Cut(tag, ",")
	return tag, tagOptions(opt)
}

// Contains reports whether our comma-separated options contains a given option.
func (o tagOptions) Contains(optionName string) bool {
	if len(o) == 0 {
		return false
	}
	s := string(o)
	for s != "" {
		var name string
		name, s, _ = strings.Cut(s, ",")
		if strings.TrimSpace(name) == optionName {
			return true
		}
	}
	return false
}
