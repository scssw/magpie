package provider

import (
	"encoding/json"
	"strings"
	"testing"
)

// mustQuote is s as a JSON string, for a body built by hand.
func mustQuote(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func TestWorkBuddyBody(t *testing.T) {
	roles := func(body []byte) (out []string) {
		var m struct {
			Messages []struct {
				Role    string `json:"role"`
				Content any    `json:"content"`
			} `json:"messages"`
		}
		if err := json.Unmarshal(body, &m); err != nil {
			t.Fatalf("%v: %s", err, body)
		}
		for _, msg := range m.Messages {
			out = append(out, msg.Role)
		}
		return out
	}

	// none: one is put first, the rest kept as they were
	in := []byte(`{"model":"deepseek-v4.1-flash","stream":true,"max_tokens":1024,"messages":[{"role":"user","content":"<b>hi</b> & 1.50"}]}`)
	out := wbBody(in)
	if got := roles(out); len(got) != 2 || got[0] != "system" || got[1] != "user" {
		t.Fatalf("roles %v: %s", got, out)
	}
	var m map[string]any
	_ = json.Unmarshal(out, &m)
	if m["model"] != "deepseek-v4.1-flash" || m["stream"] != true || m["max_tokens"] != 1024.0 {
		t.Fatalf("fields: %s", out)
	}
	if first := m["messages"].([]any)[0].(map[string]any); first["content"] != wbSystem {
		t.Fatalf("system: %v", first)
	}
	if user := m["messages"].([]any)[1].(map[string]any); user["content"] != "<b>hi</b> & 1.50" {
		t.Fatalf("user: %v", user)
	}

	// a system prompt later on still wants one first
	if got := roles(wbBody([]byte(`{"messages":[{"role":"user","content":"a"},{"role":"system","content":"s"}]}`))); len(got) != 3 || got[0] != "system" {
		t.Fatalf("late system: %v", got)
	}

	// already first, not a chat, or not JSON: the same bytes
	for _, b := range []string{
		`{"messages":[{"role":"system","content":"own"},{"role":"user","content":"a"}]}`,
		`{"messages":[]}`,
		`{"prompt":"a cat"}`,
		`{"messages":`,
	} {
		if got := string(wbBody([]byte(b))); got != b {
			t.Errorf("%s became %s", b, got)
		}
	}
}

// A system prompt WorkBuddy refuses outright is reworded, the rest of it
// kept as it came, so the agent still has its instructions (#182).
func TestWorkBuddyBodyRewordsForeignOpening(t *testing.T) {
	opening := wbForeign[0].from
	prompt := opening + "\n\n# Personality\n\nBe terse. " + opening + " is how you start."
	in := []byte(`{"model":"workbuddy/hy3","messages":[{"role":"system","content":` +
		mustQuote(prompt) + `},{"role":"user","content":"hi"}]}`)

	out := wbBody(in)
	var m struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if len(m.Messages) != 2 || m.Messages[0].Role != "system" {
		t.Fatalf("messages: %s", out)
	}
	got := m.Messages[0].Content
	// the opening is gone, both times it appeared
	if strings.Contains(got, opening) {
		t.Errorf("still refused:\n%s", got)
	}
	if !strings.Contains(got, wbForeign[0].to) {
		t.Errorf("not reworded:\n%s", got)
	}
	// and nothing else was touched
	if !strings.Contains(got, "# Personality") || !strings.Contains(got, "is how you start.") {
		t.Errorf("the rest was lost:\n%s", got)
	}
	if m.Messages[1].Content != "hi" {
		t.Errorf("user: %s", out)
	}

	// a system prompt that isn't one of those is left alone
	same := `{"messages":[{"role":"system","content":"be brief"},{"role":"user","content":"hi"}]}`
	if got := string(wbBody([]byte(same))); got != same {
		t.Errorf("%s became %s", same, got)
	}
}

// A system message in parts is reworded part by part.
func TestWorkBuddyBodyRewordsParts(t *testing.T) {
	in := []byte(`{"messages":[{"role":"system","content":[` +
		`{"type":"text","text":` + mustQuote(wbForeign[0].from) + `},` +
		`{"type":"text","text":"# Personality"}]},{"role":"user","content":"hi"}]}`)
	out := string(wbBody(in))
	if strings.Contains(out, wbForeign[0].from) {
		t.Errorf("still refused: %s", out)
	}
	if !strings.Contains(out, "# Personality") {
		t.Errorf("the rest was lost: %s", out)
	}
}
