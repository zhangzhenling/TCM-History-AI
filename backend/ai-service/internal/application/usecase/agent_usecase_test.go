package usecase

import (
	"strings"
	"testing"

	"tcm-history-ai/backend/ai-service/internal/application/dto"
	"tcm-history-ai/backend/ai-service/internal/domain/service"
)

// TestExtractJSON_BareObject verifies a plain JSON object is returned as-is.
func TestExtractJSON_BareObject(t *testing.T) {
	in := `{"a":1,"b":"x"}`
	out := extractJSON(in)
	if out != in {
		t.Errorf("expected %q, got %q", in, out)
	}
}

// TestExtractJSON_MarkdownFence verifies ```json fences are stripped.
func TestExtractJSON_MarkdownFence(t *testing.T) {
	in := "```json\n{\"task_id\":\"t1\"}\n```"
	out := extractJSON(in)
	want := `{"task_id":"t1"}`
	if out != want {
		t.Errorf("expected %q, got %q", want, out)
	}
}

// TestExtractJSON_BareFence verifies fences without a language tag are stripped.
func TestExtractJSON_BareFence(t *testing.T) {
	in := "```\n{\"a\":1}\n```"
	out := extractJSON(in)
	if out != `{"a":1}` {
		t.Errorf("expected {\"a\":1}, got %q", out)
	}
}

// TestExtractJSON_SurroundingProse verifies JSON is extracted even when
// the LLM emits prose around it.
func TestExtractJSON_SurroundingProse(t *testing.T) {
	in := `好的，这是计划：{"task_id":"t1","sub_tasks":[]}希望能帮到你。`
	out := extractJSON(in)
	want := `{"task_id":"t1","sub_tasks":[]}`
	if out != want {
		t.Errorf("expected %q, got %q", want, out)
	}
}

// TestExtractJSON_NoBraces verifies a string without any JSON object is
// returned trimmed.
func TestExtractJSON_NoBraces(t *testing.T) {
	out := extractJSON("  hello  ")
	if out != "hello" {
		t.Errorf("expected 'hello', got %q", out)
	}
}

// TestTruncate verifies the helper clips to the requested rune count and
// appends an ellipsis.
func TestTruncate(t *testing.T) {
	if out := truncate("abcdefg", 3); out != "abc..." {
		t.Errorf("expected 'abc...', got %q", out)
	}
	if out := truncate("abc", 5); out != "abc" {
		t.Errorf("expected no truncation, got %q", out)
	}
	// Chinese runes counted as runes, not bytes
	if out := truncate("中医发展史研究", 3); out != "中医发..." {
		t.Errorf("expected '中医发...', got %q", out)
	}
}

// TestSummarizeEvidence_Empty verifies the helper handles no steps.
func TestSummarizeEvidence_Empty(t *testing.T) {
	if out := summarizeEvidence(nil); out != "" {
		t.Errorf("expected empty summary, got %q", out)
	}
}

// TestSummarizeEvidence_DirectChannel verifies direct-channel steps are
// summarised with the channel name and query.
func TestSummarizeEvidence_DirectChannel(t *testing.T) {
	steps := []dto.AgentStep{
		{
			SubTaskID: "t1",
			Channel:   "direct",
			Query:     "伤寒论作者是谁",
			Result: map[string]any{
				"evidence": "[direct] no retrieval needed",
			},
		},
	}
	summary := summarizeEvidence(steps)
	if !strings.Contains(summary, "已检索证据") {
		t.Errorf("expected header, got %q", summary)
	}
	if !strings.Contains(summary, "direct") {
		t.Errorf("expected channel name in summary, got %q", summary)
	}
	if !strings.Contains(summary, "伤寒论作者是谁") {
		t.Errorf("expected query in summary, got %q", summary)
	}
}

// TestSummarizeEvidence_RagChannel verifies rag-channel steps include
// truncated chunk content.
func TestSummarizeEvidence_RagChannel(t *testing.T) {
	steps := []dto.AgentStep{
		{
			SubTaskID: "t1",
			Channel:   "rag",
			Query:     "太阳病",
			Result: map[string]any{
				"chunks": []service.RetrievedChunk{
					{ChunkID: "c1", ClassicCode: "shanghanlun", Content: "太阳病，发热而渴。"},
				},
			},
		},
	}
	summary := summarizeEvidence(steps)
	if !strings.Contains(summary, "shanghanlun") {
		t.Errorf("expected classic_code in summary, got %q", summary)
	}
	if !strings.Contains(summary, "太阳病，发热而渴") {
		t.Errorf("expected chunk content in summary, got %q", summary)
	}
}

// TestSummarizeEvidence_GraphChannel verifies graph-channel steps include
// node name and label.
func TestSummarizeEvidence_GraphChannel(t *testing.T) {
	steps := []dto.AgentStep{
		{
			SubTaskID: "t1",
			Channel:   "graph",
			Query:     "张仲景",
			Result: map[string]any{
				"nodes": []service.GraphNode{
					{UID: "p1", Label: "Person", Name: "张仲景"},
				},
			},
		},
	}
	summary := summarizeEvidence(steps)
	if !strings.Contains(summary, "张仲景") {
		t.Errorf("expected node name in summary, got %q", summary)
	}
	if !strings.Contains(summary, "Person") {
		t.Errorf("expected node label in summary, got %q", summary)
	}
}

// TestSummarizeEvidence_TruncatesLongChunks verifies chunk content longer
// than 200 runes is truncated in the summary.
func TestSummarizeEvidence_TruncatesLongChunks(t *testing.T) {
	long := strings.Repeat("中医", 200) // 400 runes
	steps := []dto.AgentStep{
		{
			SubTaskID: "t1",
			Channel:   "rag",
			Query:     "q",
			Result: map[string]any{
				"chunks": []service.RetrievedChunk{
					{ChunkID: "c1", ClassicCode: "test", Content: long},
				},
			},
		},
	}
	summary := summarizeEvidence(steps)
	if strings.Contains(summary, long) {
		t.Errorf("expected long chunk content to be truncated, but full content found in summary")
	}
	if !strings.Contains(summary, "...") {
		t.Errorf("expected ellipsis marker in summary for truncated content")
	}
}
