package discover

import "testing"

func TestMatchGlob(t *testing.T) {
	cases := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"a.go", "a.go", true},
		{"a.go", "b.go", false},
		{"*.go", "a.go", true},
		{"*.go", "dir/a.go", false},
		{"*_test.go", "a_test.go", true},
		{"a*b*c", "abc", true},
		{"a*b*c", "axxbyyc", true},
		{"a*b*c", "axxbyy", false},
		{"*", "", true},
		{"dir/*", "dir/a.go", true},
		{"dir/*", "dir/sub/a.go", false},
		{"dir/**", "dir/a.go", true},
		{"dir/**", "dir/sub/a.go", true},
		{"dir/**", "other/a.go", false},
		{"**/a.go", "a.go", true},
		{"**/a.go", "x/y/a.go", true},
		{"**/a.go", "x/y/b.go", false},
		{"internal/**/gen/*.go", "internal/gen/x.go", true},
		{"internal/**/gen/*.go", "internal/a/b/gen/x.go", true},
		{"internal/**/gen/*.go", "internal/a/b/gen/sub/x.go", false},
		{"**", "any/depth/at/all.go", true},
		{"a/**/b", "a/b", true},
		{"a/b", "a/b/c", false},
		{"a/b/c", "a/b", false},
		{"f_?.go", "f_a.go", true},
		{"f_[ab].go", "f_b.go", true},
		{"f_[ab].go", "f_c.go", false},
		{"[", "[", false},
	}
	for _, tc := range cases {
		if got := matchGlob(tc.pattern, tc.path); got != tc.want {
			t.Errorf("matchGlob(%q, %q) = %v, want %v", tc.pattern, tc.path, got, tc.want)
		}
	}
}

func TestValidPattern(t *testing.T) {
	for _, p := range []string{"a/*.go", "**/gen/**", "internal/[a-z]*/x.go", `a\*b`} {
		if err := ValidPattern(p); err != nil {
			t.Errorf("ValidPattern(%q) = %v, want nil", p, err)
		}
	}
	for _, p := range []string{"", "internal/[gen", `a/b\`, "["} {
		if err := ValidPattern(p); err == nil {
			t.Errorf("ValidPattern(%q) = nil, want an error", p)
		}
	}
}
