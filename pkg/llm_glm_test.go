package claudecode

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/tmc/langchaingo/llms"
)

// TestLLMGLM 用于验证 GLM 兼容 Anthropic 接口的调用流程是否可用。
// 参数：t 为测试上下文。
// 返回：无。
func TestLLMGLM(t *testing.T) {
	// 手动填写测试用 prompt。
	prompt := "Hello Claude"

	// 从系统环境中读取 ANTHROPIC_AUTH_TOKEN。
	authToken := os.Getenv("ANTHROPIC_AUTH_TOKEN")
	if authToken == "" {
		t.Fatalf("ANTHROPIC_AUTH_TOKEN is required")
	}

	// 组装调用所需环境变量。
	env := map[string]string{
		"ANTHROPIC_AUTH_TOKEN":                     authToken,
		"ANTHROPIC_BASE_URL":                       "https://open.bigmodel.cn/api/anthropic",
		"API_TIMEOUT_MS":                           "3000000",
		"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
		"ANTHROPIC_DEFAULT_OPUS_MODEL":             "GLM-4.7",
		"ANTHROPIC_DEFAULT_SONNET_MODEL":           "GLM-4.7",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL":            "GLM-4.5-Air",
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
