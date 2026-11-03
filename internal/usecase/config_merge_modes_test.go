package usecase

import (
	"encoding/json"
	"strings"
	"testing"
)

// markedBehaviorTable pins the shared contract of the two section merges:
// same cases, different marker alphabets. A fix in one mode must land in
// the other — add the case to both rows.
func TestMarkedSectionMergesShareBehavior(t *testing.T) {
	managedBody := "managed line 1\nmanaged line 2"
	userSeed := "seed note"
	mkTemplate := func(ms, me, us, ue string) string {
		return ms + "\n" + managedBody + "\n" + me + "\n" + us + "\n" + userSeed + "\n" + ue + "\n"
	}
	cases := []struct {
		name     string
		template func() string
		merge    func(tpl, existing []byte) []byte
		managed  string
		userMark string
	}{
		{"markdown", func() string {
			return mkTemplate(markdownManagedStart, markdownManagedEnd, markdownUserStart, markdownUserEnd)
		}, mergeMarkdownSections, managedBody, markdownUserStart},
		{"script", func() string {
			return mkTemplate(scriptManagedStart, scriptManagedEnd, scriptUserStart, scriptUserEnd)
		}, mergeScriptSections, managedBody, scriptUserStart},
	}

	for _, tc := range cases {
		t.Run(tc.name+"/fresh inherits seed", func(t *testing.T) {
			got := string(tc.merge([]byte(tc.template()), nil))
			if !strings.Contains(got, tc.managed) || !strings.Contains(got, userSeed) {
				t.Errorf("fresh merge must carry managed + seed, got:\n%s", got)
			}
		})
		t.Run(tc.name+"/user block survives", func(t *testing.T) {
			tpl := tc.template()
			existing := tpl + ""
			// Simulate a converged-then-edited file: take a merged result
			// and append operator content to its user block.
			merged := string(tc.merge([]byte(tpl), nil))
			edited := strings.Replace(merged, userSeed, userSeed+"\noperator line", 1)
			_ = existing
			got := string(tc.merge([]byte(tpl), []byte(edited)))
			if !strings.Contains(got, "operator line") || !strings.Contains(got, tc.managed) {
				t.Errorf("user content must survive, managed must stay current, got:\n%s", got)
			}
		})
		t.Run(tc.name+"/legacy file preserved whole", func(t *testing.T) {
			legacy := "my old note\nwithout markers\n"
			got := string(tc.merge([]byte(tc.template()), []byte(legacy)))
			if !strings.Contains(got, "my old note") || !strings.Contains(got, tc.managed) {
				t.Errorf("legacy content must be kept whole next to managed, got:\n%s", got)
			}
		})
	}
}

func TestMergeScriptSectionsMarkersAreComments(t *testing.T) {
	for _, m := range []string{scriptManagedStart, scriptManagedEnd, scriptUserStart, scriptUserEnd} {
		if !strings.HasPrefix(strings.TrimSpace(m), "#") {
			t.Errorf("script marker %q must be a # comment (file stays executable)", m)
		}
	}
}

func TestDedupeUnmarkedCopy(t *testing.T) {
	managed := "line one\nline two"
	if !dedupeUnmarkedCopy(managed, "line one\nline two\n") {
		t.Error("identical modulo trailing newline must dedupe")
	}
	if dedupeUnmarkedCopy(managed, "line one\nline two\noperator edit\n") {
		t.Error("edited copy must NOT dedupe (user content would be lost)")
	}
	if dedupeUnmarkedCopy(managed, "") {
		t.Error("empty existing must not dedupe")
	}
}

func TestSubtractManagedCopy(t *testing.T) {
	managed := "managed a\nmanaged b"
	if rest, ok := subtractManagedCopy(managed, managed+"\nmy note\n"); !ok || rest != "my note" {
		t.Errorf("appended user content must survive alone, got %q (ok=%v)", rest, ok)
	}
	if rest, ok := subtractManagedCopy(managed, "my note\n"+managed); !ok || rest != "my note" {
		t.Errorf("prepended user content must survive alone, got %q (ok=%v)", rest, ok)
	}
	if _, ok := subtractManagedCopy(managed, "managed a\nEDITED\nmy note\n"); ok {
		t.Error("in-place edit of managed text must NOT subtract (keep whole, trim manually)")
	}
	if _, ok := subtractManagedCopy("", "anything"); ok {
		t.Error("empty managed section must never subtract")
	}
}

func TestMergeJSONDeepDestinationWins(t *testing.T) {
	template := []byte(`{"keep": "tpl", "nested": {"a": 1, "b": 2}, "list": [1], "gone": true}`)
	live := []byte(`{"keep": "mine", "nested": {"b": 20, "c": 30}, "list": [9], "mine": true}`)
	out, err := mergeJSONDeep(template, live)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("output must be valid JSON: %v", err)
	}
	if got["keep"] != "mine" {
		t.Errorf("scalar conflict: destination wins, got %v", got["keep"])
	}
	nested := got["nested"].(map[string]any)
	if nested["a"] != float64(1) || nested["b"] != float64(20) || nested["c"] != float64(30) {
		t.Errorf("objects merge recursively with destination winning, got %v", nested)
	}
	if len(got["list"].([]any)) != 1 || got["list"].([]any)[0] != float64(9) {
		t.Errorf("arrays keep the live value, got %v", got["list"])
	}
	if got["mine"] != true || got["gone"] != true {
		t.Errorf("both sides' unique keys survive, got %v", got)
	}
}

func TestMergeJSONDeepIdempotent(t *testing.T) {
	template := []byte(`{"a": {"x": 1}, "list": [1, 2]}`)
	live := []byte(`{"a": {"y": 2}, "list": [3]}`)
	once, err := mergeJSONDeep(template, live)
	if err != nil {
		t.Fatal(err)
	}
	twice, err := mergeJSONDeep(template, once)
	if err != nil {
		t.Fatal(err)
	}
	if string(once) != string(twice) {
		t.Errorf("merge must be idempotent:\n%s\nvs\n%s", once, twice)
	}
}

func TestMergeJSONDeepRejectsNonObjects(t *testing.T) {
	if _, err := mergeJSONDeep([]byte(`[1,2]`), []byte(`{"a":1}`)); err == nil {
		t.Error("non-object template must fail (caller keeps live untouched)")
	}
	if _, err := mergeJSONDeep([]byte(`{"a":1}`), []byte(`not json`)); err == nil {
		t.Error("unparseable live file must fail (caller keeps live untouched)")
	}
}

func TestMergePermissionsUnion(t *testing.T) {
	template := []byte(`{"permissions": {"allow": ["Edit", "Write"], "ask": ["Shell(rm:*)"], "deny": ["Edit(.git/**)"]}}`)
	live := []byte(`{"permissions": {"allow": ["Write", "Shell(custom:*)"], "ask": [], "deny": ["Edit(.git/**)"]}}`)
	out, err := mergePermissionsUnion(template, live)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	var got permissionsShape
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("output must decode: %v", err)
	}
	wantAllow := []string{"Edit", "Write", "Shell(custom:*)"}
	if strings.Join(got.Permissions.Allow, ",") != strings.Join(wantAllow, ",") {
		t.Errorf("allow must union template-first, got %v", got.Permissions.Allow)
	}
	if len(got.Permissions.Ask) != 1 || len(got.Permissions.Deny) != 1 {
		t.Errorf("ask/deny survive, got %+v", got.Permissions)
	}
}

func TestMergeMarkdownPreservesFrontmatter(t *testing.T) {
	template := "---\nname: x\ndescription: y\n---\n" +
		markdownManagedStart + "\nmanaged body\n" + markdownManagedEnd + "\n" +
		markdownUserStart + "\nseed\n" + markdownUserEnd + "\n"
	existing := "---\nname: x\ndescription: y\n---\n" +
		markdownManagedStart + "\nold managed\n" + markdownManagedEnd + "\n" +
		markdownUserStart + "\nmy note\n" + markdownUserEnd + "\n"
	got := string(mergeMarkdownSections([]byte(template), []byte(existing)))
	if !strings.HasPrefix(got, "---\nname: x\ndescription: y\n---\n") {
		t.Errorf("frontmatter must stay verbatim at byte zero, got:\n%s", got)
	}
	for _, want := range []string{"managed body", "my note", markdownManagedStart, markdownUserEnd} {
		if !strings.Contains(got, want) {
			t.Errorf("merged file must contain %q, got:\n%s", want, got)
		}
	}
	if strings.Contains(got, "old managed") {
		t.Errorf("stale managed block must be replaced, got:\n%s", got)
	}
}

func TestSplitFrontmatter(t *testing.T) {
	header, rest := splitFrontmatter("---\na: 1\n---\nbody\n")
	if header != "---\na: 1\n---\n" || rest != "body\n" {
		t.Errorf("got header %q rest %q", header, rest)
	}
	if h, r := splitFrontmatter("# plain\nbody\n"); h != "" || r != "# plain\nbody\n" {
		t.Errorf("no frontmatter must pass through, got %q %q", h, r)
	}
	if h, r := splitFrontmatter("---\nunclosed\nbody\n"); h != "" {
		t.Errorf("unclosed frontmatter is not a header, got %q %q", h, r)
	}
}

func TestMergeScriptSectionsPowerShellProfile(t *testing.T) {
	template := scriptManagedStart + "\n$env:FOO = '1'\n" + scriptManagedEnd + "\n" +
		scriptUserStart + "\n" + scriptUserEnd + "\n"
	live := "$env:FOO = '1'\nfunction prompt { 'mine> ' }\n"
	got := string(mergeScriptSections([]byte(template), []byte(live)))
	if !strings.Contains(got, "function prompt { 'mine> ' }") {
		t.Errorf("operator profile content must survive, got:\n%s", got)
	}
	// Idempotent: merging the merged result changes nothing.
	again := string(mergeScriptSections([]byte(template), []byte(got)))
	if again != got {
		t.Errorf("script merge must be idempotent:\n%s\nvs\n%s", got, again)
	}
}

func TestMergePermissionsUnionRejectsOtherShapes(t *testing.T) {
	if _, err := mergePermissionsUnion([]byte(`{"other": 1}`), []byte(`{"permissions": {"allow": []}}`)); err == nil {
		t.Error("template without permissions shape must fail")
	}
	if _, err := mergePermissionsUnion([]byte(`{"permissions": {"allow": []}}`), []byte(`garbage`)); err == nil {
		t.Error("unparseable live file must fail (kept untouched)")
	}
}
