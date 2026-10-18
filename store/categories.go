package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// categoriesFile lists category folders that exist even when no model is placed
// in them yet (a category otherwise exists only through its models' meta).
const categoriesFile = "_categories.json"

// Uncategorized is the virtual root bucket for models without a category.
const Uncategorized = "Uncategorized"

type categoriesDoc struct {
	Categories [][]string `json:"categories"`
}

// LoadCategories returns the persisted (possibly empty) category paths.
func LoadCategories(dir string) [][]string {
	data, err := os.ReadFile(filepath.Join(dir, categoriesFile))
	if err != nil {
		return nil
	}
	var d categoriesDoc
	if json.Unmarshal(data, &d) != nil {
		return nil
	}
	return d.Categories
}

// SaveCategories persists the category paths.
func SaveCategories(dir string, cats [][]string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if cats == nil {
		cats = [][]string{}
	}
	data, err := json.MarshalIndent(categoriesDoc{Categories: cats}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, categoriesFile), append(data, '\n'), 0o644)
}

// ValidCategoryName checks one path segment (and trims it).
func ValidCategoryName(s string) (string, error) {
	s = strings.TrimSpace(s)
	switch {
	case s == "":
		return "", fmt.Errorf("category name is empty")
	case len(s) > 60:
		return "", fmt.Errorf("category name is too long (max 60 characters)")
	case strings.ContainsAny(s, "/\\›"):
		return "", fmt.Errorf("category name must not contain / \\ or ›")
	case strings.EqualFold(s, Uncategorized):
		return "", fmt.Errorf("%q is reserved for models without a category", Uncategorized)
	}
	return s, nil
}

// CleanCategoryPath validates and trims every segment of a path.
func CleanCategoryPath(path []string) ([]string, error) {
	out := make([]string, 0, len(path))
	for _, seg := range path {
		c, err := ValidCategoryName(seg)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

var aliasRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

// NormalizeAlias lower-cases and validates an alias (used for --alias and
// configs/<alias>.json, so it must be filesystem- and URL-safe).
func NormalizeAlias(s string) (string, error) {
	a := strings.ToLower(strings.TrimSpace(s))
	if !aliasRe.MatchString(a) {
		return "", fmt.Errorf("alias %q is invalid: use lowercase letters, digits, '.', '_' or '-' (max 64, starting with a letter or digit)", s)
	}
	return a, nil
}

// PathHasPrefix reports whether category path c lies inside (or equals) prefix p.
func PathHasPrefix(c, p []string) bool {
	if len(c) < len(p) {
		return false
	}
	for i := range p {
		if c[i] != p[i] {
			return false
		}
	}
	return true
}

// RenamePathPrefix returns c with the segment at index len(from)-1 replaced by
// newName, provided c lies inside from; otherwise c is returned unchanged.
func RenamePathPrefix(c, from []string, newName string) []string {
	if len(from) == 0 || !PathHasPrefix(c, from) {
		return c
	}
	out := append([]string(nil), c...)
	out[len(from)-1] = newName
	return out
}

// PathEqual reports whether two category paths are identical.
func PathEqual(a, b []string) bool {
	return len(a) == len(b) && PathHasPrefix(a, b)
}
