package store

import (
	"reflect"
	"testing"
)

func TestCategoryNameAndAliasValidation(t *testing.T) {
	for _, bad := range []string{"", "  ", "a/b", `a\b`, "a›b", "uncategorized", "UNCATEGORIZED"} {
		if _, err := ValidCategoryName(bad); err == nil {
			t.Errorf("ValidCategoryName(%q) should fail", bad)
		}
	}
	if got, err := ValidCategoryName("  Qwen 3.8 "); err != nil || got != "Qwen 3.8" {
		t.Errorf("trim: %q %v", got, err)
	}
	for in, want := range map[string]string{"My-Model.v2": "my-model.v2", " a_b ": "a_b"} {
		if got, err := NormalizeAlias(in); err != nil || got != want {
			t.Errorf("NormalizeAlias(%q) = %q, %v", in, got, err)
		}
	}
	for _, bad := range []string{"", "-lead", "has space", "ünï", "a/b"} {
		if _, err := NormalizeAlias(bad); err == nil {
			t.Errorf("NormalizeAlias(%q) should fail", bad)
		}
	}
}

func TestCategoryPathHelpers(t *testing.T) {
	c := []string{"Qwen", "3.8", "x"}
	if !PathHasPrefix(c, []string{"Qwen", "3.8"}) || PathHasPrefix(c, []string{"Qwen", "4"}) || PathHasPrefix([]string{"Qwen"}, c) {
		t.Error("PathHasPrefix wrong")
	}
	if got := RenamePathPrefix(c, []string{"Qwen", "3.8"}, "4.0"); !reflect.DeepEqual(got, []string{"Qwen", "4.0", "x"}) {
		t.Errorf("RenamePathPrefix = %v", got)
	}
	if got := RenamePathPrefix(c, []string{"Other"}, "Z"); !reflect.DeepEqual(got, c) {
		t.Errorf("unrelated path must be unchanged: %v", got)
	}
	if &RenamePathPrefix(c, []string{"Qwen"}, "Z")[0] == &c[0] {
		t.Error("RenamePathPrefix must not alias its input")
	}
}

func TestCategoriesFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if got := LoadCategories(dir); got != nil {
		t.Errorf("missing file: %v", got)
	}
	want := [][]string{{"A"}, {"A", "B"}}
	if err := SaveCategories(dir, want); err != nil {
		t.Fatal(err)
	}
	if got := LoadCategories(dir); !reflect.DeepEqual(got, want) {
		t.Errorf("round trip: %v", got)
	}
}
