package claudecode

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/tmc/langchaingo/llms"
)

// TestLLMDeepSeek 用于验证 DeepSeek 兼容 Anthropic 接口的调用流程是否可用。
// 参数：t 为测试上下文。
// 返回：无。
func TestLLMDeepSeek(t *testing.T) {
	// 手动填写测试用 prompt。
	prompt := "Hello DeepSeek"

	// 从系统环境中读取 ANTHROPIC_AUTH_TOKEN，其他参数使用默认值。
	authToken := os.Getenv("ANTHROPIC_AUTH_TOKEN")
	if authToken == "" {
		t.Fatalf("ANTHROPIC_AUTH_TOKEN is required")
	}

	// 使用 DeepSeek 默认环境变量配置。
	env := map[string]string{
		"ANTHROPIC_AUTH_TOKEN": authToken,
		"ANTHROPIC_BASE_URL":   "https://api.deepseek.com/anthropic",
		"API_TIMEOUT_MS":       "600000",
		"ANTHROPIC_MODEL":      "deepseek-chat",
		"ANTHROPIC_SMALL_FAST_MODEL": "deepseek-chat",
		"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
	}

	// 手动填写 option 参数。
	opts := []Option{
		WithModel(""),
		WithSystemPrompt(""),
		WithCLIPath(""),
		WithPermissionMode("bypassPermissions"),
		WithCwd(""),
		WithEnv(env),
	}

	// 初始化 Claude Code LLM。
	llm, err := New(opts...)
	if err != nil {
		t.Fatalf("init claudecode: %v", err)
	}

	// 调用模型生成响应。
	ctx := context.Background()
	fmt.Printf("[start] prompt: %s\n\n", prompt)
	streamingFunc := func(_ context.Context, chunk []byte) error {
		// 元信息输出到 stderr，内容输出到 stdout。
		timestamp := time.Now().Format(time.RFC3339Nano)
		fmt.Fprintf(os.Stderr, "[chunk] ts=%s len=%d\n", timestamp, len(chunk))
		fmt.Print(string(chunk))
		return nil
	}
	resp, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt, llms.WithStreamingFunc(streamingFunc))
	if err != nil {
		t.Fatalf("claudecode error: %v", err)
	}
	fmt.Println("\n\n[end] done")

	if resp == "" {
		t.Fatalf("empty response")
	}
}
