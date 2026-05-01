package game

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Wordpack contains parsed bundled wordpack metadata and words.
type Wordpack struct {
	ID    string
	Label string
	Words []string
}

// ParseWordpack parses non-empty, non-comment wordpack lines.
func ParseWordpack(content string) []string {
	lines := strings.Split(content, "\n")
	words := make([]string, 0, len(lines))
	for _, line := range lines {
		word := strings.TrimSpace(line)
		if word == "" || strings.HasPrefix(word, "#") {
			continue
		}
		words = append(words, word)
	}
	return words
}

// LoadWordpacks loads all .txt wordpacks from dir keyed by filename without extension.
func LoadWordpacks(dir string) (map[string]Wordpack, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read wordpack dir: %w", err)
	}
	packs := map[string]Wordpack{}
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".txt" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		bytes, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read wordpack %s: %w", path, err)
		}
		id := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		packs[id] = Wordpack{ID: id, Label: labelForWordpack(id), Words: ParseWordpack(string(bytes))}
	}
	return packs, nil
}

func labelForWordpack(id string) string {
	parts := strings.FieldsFunc(id, func(r rune) bool { return r == '-' || r == '_' })
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func uniqueWords(words []string) []string {
	seen := map[string]bool{}
	unique := make([]string, 0, len(words))
	for _, word := range words {
		word = strings.TrimSpace(word)
		if word == "" || seen[word] {
			continue
		}
		seen[word] = true
		unique = append(unique, word)
	}
	sort.Strings(unique)
	return unique
}
