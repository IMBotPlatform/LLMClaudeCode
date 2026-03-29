package claudecode

import "testing"

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
