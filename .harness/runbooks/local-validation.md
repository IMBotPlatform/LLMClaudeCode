# Runbook: Local Validation

## Default Check

```bash
go test ./...
```

## Preconditions

- 本机可找到 `claude` CLI
- 设置 `ANTHROPIC_AUTH_TOKEN`

## Integration Test Notes

- `pkg/llm_glm_test.go` 使用 GLM 兼容 Anthropic 接口
- `pkg/llm_ds_test.go` 使用 DeepSeek 兼容 Anthropic 接口
- 两者都依赖外部网络与有效 token

## Downstream Reminder

- 如改动 session 或 Option 语义，检查 `wechataibot/internal/app/llm_loader.go`
