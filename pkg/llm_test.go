package claudecode

import (
	"context"
	"strings"
	"testing"
)

func TestShouldInsertAssistantParagraphBreak(t *testing.T) {
	tests := []struct {
		name     string
		existing string
		next     string
		want     bool
	}{
		{
			name:     "empty existing",
			existing: "",
			next:     "第二段",
			want:     false,
		},
		{
			name:     "empty next",
			existing: "第一段",
			next:     "",
			want:     false,
		},
		{
			name:     "existing ends with newline",
			existing: "第一段\n",
			next:     "第二段",
			want:     false,
		},
		{
			name:     "next starts with newline",
			existing: "第一段",
			next:     "\n第二段",
			want:     false,
		},
		{
			name:     "insert paragraph break",
			existing: "第一段",
			next:     "第二段",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldInsertAssistantParagraphBreak(tt.existing, tt.next)
			if got != tt.want {
				t.Fatalf("shouldInsertAssistantParagraphBreak(%q, %q) = %v, want %v", tt.existing, tt.next, got, tt.want)
			}
		})
	}
}

func TestExtractAssistantContentIncludesThinkingBlocks(t *testing.T) {
	payload := map[string]any{
		"message": map[string]any{
			"content": []any{
				map[string]any{"type": "thinking", "thinking": "先分析一下"},
				map[string]any{"type": "text", "text": "这是结果"},
			},
		},
	}

	blocks, err := extractAssistantContent(payload)
	if err != nil {
		t.Fatalf("extractAssistantContent: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("unexpected block count: %d", len(blocks))
	}
	if blocks[0].Kind != assistantContentThinking || blocks[0].Text != "先分析一下" {
		t.Fatalf("unexpected first block: %+v", blocks[0])
	}
	if blocks[1].Kind != assistantContentText || blocks[1].Text != "这是结果" {
		t.Fatalf("unexpected second block: %+v", blocks[1])
	}
}

func TestReadStreamThinkingTagsDisabledByDefault(t *testing.T) {
	llm := &LLM{}
	stdout := strings.NewReader(`{"type":"assistant","message":{"content":[{"type":"thinking","thinking":"分析过程"},{"type":"text","text":"最终答案"}]}}` + "\n")

	got, _, err := llm.readStream(context.Background(), stdout, nil)
	if err != nil {
		t.Fatalf("readStream: %v", err)
	}
	if got != "最终答案" {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestReadStreamThinkingTagsEnabled(t *testing.T) {
	llm := &LLM{
		opts: Options{
			ThinkingTags: true,
		},
	}
	stdout := strings.NewReader(`{"type":"assistant","message":{"content":[{"type":"thinking","thinking":"分析过程"},{"type":"text","text":"最终答案"}]}}` + "\n")

	got, _, err := llm.readStream(context.Background(), stdout, nil)
	if err != nil {
		t.Fatalf("readStream: %v", err)
	}
	want := "<think>分析过程</think>\n\n最终答案"
	if got != want {
		t.Fatalf("unexpected output: got %q want %q", got, want)
	}
}
