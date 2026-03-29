# LLMClaudeCode

基于 Claude Code CLI 的 Go 语言适配器，提供 `llms.Model` 实现，便于接入 Langchaingo 生态。

## 功能特点

- 通过 `claude` CLI 调用 Claude Code
- 解析 `--output-format stream-json` 输出
- 支持 `thinking` / `tool_use` / `tool_result` 事件解析
- 支持 `OutputMode`、`WithThinkingTags`、session 恢复相关 Option
- 实现 `llms.Model` 接口，兼容 `chains/agents`
- 默认 permission mode: `bypassPermissions`

## 快速使用

```go
import (
    "context"

    "github.com/IMBotPlatform/LLMClaudeCode/pkg"
    "github.com/tmc/langchaingo/llms"
)

llm, err := claudecode.New(
    claudecode.WithPermissionMode("bypassPermissions"),
)
if err != nil {
    // handle error
}

resp, err := llms.GenerateFromSinglePrompt(context.Background(), llm, "Hello")
```

## 集成测试

GLM 与 DeepSeek 的兼容接口测试都读取 `ANTHROPIC_AUTH_TOKEN`：

```bash
ANTHROPIC_AUTH_TOKEN=你的密钥 go test -v -run TestLLMGLM -count=1 ./pkg
ANTHROPIC_AUTH_TOKEN=你的密钥 go test -v -run TestLLMDeepSeek -count=1 ./pkg
```

说明：
- 流式内容输出到 stdout
- chunk 元信息（时间戳/长度）输出到 stderr
- `pkg/llm_test.go` 还包含不依赖外部 token 的 stream parsing / thinking tag 单元测试
