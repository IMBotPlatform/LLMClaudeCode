# Runbook: Local Validation

## Default Check

```bash
go test ./...
```

## Preconditions

- 本机可找到 `claude` CLI
- 设置 `ANTHROPIC_AUTH_TOKEN`

## Targeted Checks

```bash
go test ./pkg -run 'Test(ShouldInsertAssistantParagraphBreak|ExtractAssistantContentIncludesThinkingBlocks|ReadStreamThinkingTags.*)'
```

```bash
ANTHROPIC_AUTH_TOKEN=<token> go test -v -run 'TestLLM(GLM|DeepSeek)' -count=1 ./pkg
```

## Integration Test Notes

- `pkg/llm_glm_test.go` 使用 GLM 兼容 Anthropic 接口
- `pkg/llm_ds_test.go` 使用 DeepSeek 兼容 Anthropic 接口
- 两者都依赖外部网络与有效 token

## Observed Conflict

- `README.md` 的 GLM 测试示例使用 `AUTH_TOKEN`
- 实际测试代码读取的是 `ANTHROPIC_AUTH_TOKEN`
- 当前 harness 以 `pkg/llm_glm_test.go`、`pkg/llm_ds_test.go` 为准

## Downstream Reminder

- 如改动 session、thinking tag 或 Option 语义，检查 `wechataibot/internal/app/llm_loader.go`
