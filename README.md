# LLMClaudeCode

基于 Claude Code CLI 的 Go 语言适配器，提供 `llms.Model` 实现，便于接入 Langchaingo 生态。

## 功能特点

- 通过 `claude` CLI 调用 Claude Code
- 解析 `--output-format stream-json` 输出
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

## GLM 测试

设置 `AUTH_TOKEN` 后运行测试：

```bash
AUTH_TOKEN=你的密钥 go test -v -run TestLLMGLM -count=1 ./pkg
```

说明：
- 流式内容输出到 stdout
- chunk 元信息（时间戳/长度）输出到 stderr
