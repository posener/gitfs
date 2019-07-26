package glob

import (
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// Patterns can glob-match files or directories.
type Patterns []string

// New returns a new glob pattern. It returns an error if any of the
// patterns is invalid.
func New(patterns ...string) (Patterns, error) {
	if err := checkPatterns(patterns); err != nil {
		return nil, err
	}
	return Patterns(patterns), nil
}

// Match a path to the defined patterns. If it is a file a full match
// is required. If it is a directory, only matching a prefix of any of
// the patterns is required.
func (p Patterns) Match(path string, isDir bool) bool {
	path = filepath.Clean(path)
	return (isDir && p.matchPrefix(path)) || (!isDir && p.matchFull(path))
}

// matchFull finds a matching of the whole name to any of the patterns.
func (p Patterns) matchFull(name string) bool {
	for _, pattern := range p {
		if ok, _ := filepath.Match(pattern, name); ok {
			return true
		}
	}
	return false
}

// matchPrefix finds a matching of prefix to a prefix of any of the patterns.
func (p Patterns) matchPrefix(prefix string) bool {
	parts := strings.Split(prefix, string(filepath.Separator))
nextPattern:
	for _, pattern := range p {
		patternParts := strings.Split(pattern, string(filepath.Separator))
		if len(patternParts) < len(parts) {
			continue
		}
		for i := 0; i < len(parts); i++ {
			if ok, _ := filepath.Match(patternParts[i], parts[i]); !ok {
				continue nextPattern
			}
		}
		return true
	}
	return false
}

// checkPattens checks the validity of the patterns.
func checkPatterns(patterns []string) error {
	var badPatterns []string
	for _, pattern := range patterns {
		_, err := filepath.Match(pattern, "x")
		if err != nil {
			badPatterns = append(badPatterns, pattern)
			return errors.Wrap(err, pattern)
		}
	}
	if len(badPatterns) > 0 {
		return errors.Wrap(filepath.ErrBadPattern, strings.Join(badPatterns, ", "))
	}
	return nil
}
