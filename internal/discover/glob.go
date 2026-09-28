package discover

import (
	"errors"
	"fmt"
	"path"
	"strings"
)

// matchGlob reports whether a slash-separated path matches a pattern.
// Within one segment the syntax is that of path.Match.
// A "**" segment matches any number of whole segments, including none.
func matchGlob(pattern, rel string) bool {
	return matchSegments(strings.Split(pattern, "/"), strings.Split(rel, "/"))
}

func matchSegments(pattern, segments []string) bool {
	for len(pattern) > 0 {
		if pattern[0] == "**" {
			for skip := 0; skip <= len(segments); skip++ {
				if matchSegments(pattern[1:], segments[skip:]) {
					return true
				}
			}
			return false
		}
		if len(segments) == 0 || !matchSegment(pattern[0], segments[0]) {
			return false
		}
		pattern, segments = pattern[1:], segments[1:]
	}
	return len(segments) == 0
}

// ValidPattern returns an error when matchGlob cannot use a pattern, because matchGlob treats a malformed pattern as matching nothing.
func ValidPattern(pattern string) error {
	if pattern == "" {
		return errors.New("pattern is empty")
	}
	for _, segment := range strings.Split(pattern, "/") {
		// Match checks the whole pattern for syntax, even when the name does not match.
		if _, err := path.Match(segment, ""); err != nil {
			return fmt.Errorf("pattern %q: %w", pattern, err)
		}
	}
	return nil
}

// matchSegment matches one segment with the syntax of path.Match, and a malformed pattern matches nothing.
func matchSegment(pattern, name string) bool {
	ok, err := path.Match(pattern, name)
	return ok && err == nil
}
