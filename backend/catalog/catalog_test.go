package catalog

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

const sourceRoot = "../../catalog"

// validSources is enforced so a typo'd parent dir (e.g. "parkrvcp/") becomes
// a test failure rather than silently producing a game with an unknown source.
var validSources = map[string]bool{
	"parkervcp":    true,
	"pelican-eggs": true,
	"linuxgsm":     true,
}

var validTiers = map[string]bool{
	"supported":    true,
	"experimental": true,
	"disabled":     true,
}

func TestSourceFilesValidate(t *testing.T) {
	seen := map[string]string{}
	err := filepath.WalkDir(sourceRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(p, ".json") {
			return nil
		}
		rel, _ := filepath.Rel(sourceRoot, p)
		parts := strings.SplitN(rel, string(filepath.Separator), 2)
		if len(parts) != 2 {
			return nil
		}
		source := parts[0]
		if !validSources[source] {
			t.Errorf("%s: unknown source dir %q (must be one of %v)", rel, source, sortedKeys(validSources))
			return nil
		}
		fileID := strings.TrimSuffix(path.Base(p), ".json")

		data, err := os.ReadFile(p)
		if err != nil {
			t.Errorf("%s: read: %v", rel, err)
			return nil
		}
		var g Game
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&g); err != nil {
			t.Errorf("%s: parse: %v", rel, err)
			return nil
		}
		if g.ID != fileID {
			t.Errorf("%s: id %q must match filename", rel, g.ID)
		}
		if g.Name == "" {
			t.Errorf("%s: name required", rel)
		}
		if !validTiers[g.Tier] {
			t.Errorf("%s: tier %q must be supported|experimental|disabled", rel, g.Tier)
		}
		switch g.Tier {
		case "disabled":
			if g.DisabledReason == "" {
				t.Errorf("%s: tier=disabled requires disabledReason", rel)
			}
		case "supported", "experimental":
			if g.InstallRecipe == nil {
				t.Errorf("%s: tier=%s requires installRecipe", rel, g.Tier)
			} else if g.InstallRecipe.Method == "" {
				t.Errorf("%s: installRecipe.method required", rel)
			}
		}
		if other, dup := seen[g.ID]; dup {
			t.Errorf("duplicate id %q: %s and %s", g.ID, other, rel)
		}
		seen[g.ID] = rel
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
