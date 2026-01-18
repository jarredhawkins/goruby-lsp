package index

import (
	"bufio"
	"os"
	"regexp"
	"strings"
	"sync"
)

// TrigramIndex provides text search across the codebase
// Uses a simple inverted index for now, can be replaced with Zoekt later
type TrigramIndex struct {
	mu sync.RWMutex

	// Inverted index: trigram -> set of file paths
	trigrams map[string]map[string]struct{}

	// File content cache for verification
	files map[string]string
}

// NewTrigramIndex creates a new trigram index
func NewTrigramIndex() *TrigramIndex {
	return &TrigramIndex{
		trigrams: make(map[string]map[string]struct{}),
		files:    make(map[string]string),
	}
}

// AddFile indexes a file's content
func (t *TrigramIndex) AddFile(path string, content []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()

	contentStr := string(content)
	t.files[path] = contentStr

	// Extract trigrams
	for i := 0; i <= len(contentStr)-3; i++ {
		tri := contentStr[i : i+3]
		if t.trigrams[tri] == nil {
			t.trigrams[tri] = make(map[string]struct{})
		}
		t.trigrams[tri][path] = struct{}{}
	}
}

// RemoveFile removes a file from the index
func (t *TrigramIndex) RemoveFile(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	content, ok := t.files[path]
	if !ok {
		return
	}

	delete(t.files, path)

	// Remove trigrams
	for i := 0; i <= len(content)-3; i++ {
		tri := content[i : i+3]
		if files, ok := t.trigrams[tri]; ok {
			delete(files, path)
			if len(files) == 0 {
				delete(t.trigrams, tri)
			}
		}
	}
}

// Search finds references to the given pattern
func (t *TrigramIndex) Search(pattern string) []*Reference {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Find candidate files using trigrams
	candidates := t.findCandidates(pattern)
	if len(candidates) == 0 {
		return nil
	}

	// Build word boundary regex for verification
	pinfo := buildPatternInfo(pattern)

	var refs []*Reference

	for path := range candidates {
		content, ok := t.files[path]
		if !ok {
			continue
		}

		// Verify matches line by line
		lineRefs := t.searchInContentWithInfo(path, content, pinfo, len(pattern))
		refs = append(refs, lineRefs...)
	}

	return refs
}

// findCandidates uses trigram intersection to find candidate files
func (t *TrigramIndex) findCandidates(pattern string) map[string]struct{} {
	if len(pattern) < 3 {
		// Too short for trigrams, return all files
		result := make(map[string]struct{})
		for path := range t.files {
			result[path] = struct{}{}
		}
		return result
	}

	var candidates map[string]struct{}

	for i := 0; i <= len(pattern)-3; i++ {
		tri := pattern[i : i+3]
		files, ok := t.trigrams[tri]
		if !ok {
			// Trigram not found, no matches
			return nil
		}

		if candidates == nil {
			// First trigram
			candidates = make(map[string]struct{})
			for path := range files {
				candidates[path] = struct{}{}
			}
		} else {
			// Intersect with existing candidates
			for path := range candidates {
				if _, ok := files[path]; !ok {
					delete(candidates, path)
				}
			}
		}

		if len(candidates) == 0 {
			return nil
		}
	}

	return candidates
}

// searchInContent finds all matches in file content
func (t *TrigramIndex) searchInContent(path, content string, pattern *regexp.Regexp) []*Reference {
	return t.searchInContentWithInfo(path, content, patternInfo{regex: pattern, endsWithSpecial: false}, 0)
}

// searchInContentWithInfo finds all matches with correct length handling for Ruby methods
func (t *TrigramIndex) searchInContentWithInfo(path, content string, pinfo patternInfo, patternLen int) []*Reference {
	var refs []*Reference

	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		matches := pinfo.regex.FindAllStringIndex(line, -1)
		for _, match := range matches {
			length := match[1] - match[0]
			// If pattern ends with ? ! =, the regex includes an extra char - use original length
			if pinfo.endsWithSpecial && patternLen > 0 {
				length = patternLen
			}
			refs = append(refs, &Reference{
				FilePath: path,
				Line:     lineNum,
				Column:   match[0],
				Length:   length,
				LineText: line,
			})
		}
	}

	return refs
}

// SearchFile searches for references in a specific file
func (t *TrigramIndex) SearchFile(path, pattern string) []*Reference {
	t.mu.RLock()
	defer t.mu.RUnlock()

	content, ok := t.files[path]
	if !ok {
		// Try reading from disk
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content = string(data)
	}

	wordPattern := buildWordBoundaryPattern(pattern)
	return t.searchInContent(path, content, wordPattern)
}

// rubyMethodSuffix tracks if a pattern ends with Ruby method suffix
type patternInfo struct {
	regex           *regexp.Regexp
	endsWithSpecial bool // ends with ? ! or =
}

// buildWordBoundaryPattern creates a regex that properly handles Ruby method names
// ending in ? ! or = which can't use \b at the end
func buildWordBoundaryPattern(pattern string) *regexp.Regexp {
	return buildPatternInfo(pattern).regex
}

func buildPatternInfo(pattern string) patternInfo {
	escapedPattern := regexp.QuoteMeta(pattern)
	var regexPattern string
	endsWithSpecial := false
	if len(pattern) > 0 {
		lastChar := pattern[len(pattern)-1]
		if lastChar == '?' || lastChar == '!' || lastChar == '=' {
			regexPattern = `\b` + escapedPattern + `(?:[^a-zA-Z0-9_]|$)`
			endsWithSpecial = true
		} else {
			regexPattern = `\b` + escapedPattern + `\b`
		}
	} else {
		regexPattern = `\b` + escapedPattern + `\b`
	}
	return patternInfo{
		regex:           regexp.MustCompile(regexPattern),
		endsWithSpecial: endsWithSpecial,
	}
}
