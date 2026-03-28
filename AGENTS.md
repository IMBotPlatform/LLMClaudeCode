# AGENTS.md

## Mission / Scope

这个仓库负责把 Claude Code CLI 包装成 `langchaingo/llms.Model`。

- owns: CLI 发现、命令构造、`stream-json` 解析、Option 体系、session 参数映射
- not owns: WeCom 协议、命令系统、产品部署与 skills

## Start Here

1. `.harness/README.md`
2. `.harness/generated/repo-manifest.yaml`
3. `.harness/generated/api-index.md`
4. `README.md`

## Source of Truth

- `pkg/llm.go`
- `pkg/options.go`
- `pkg/llm_glm_test.go`
- `pkg/llm_ds_test.go`

## Important Directories

- `pkg/`
- `changelogs/`

## Hard Constraints

- 保持 `llms.Model` 语义可用
- 不把产品特定 env 规则硬编码成唯一模式
- 改动 Option 或 session 语义时，需检查下游 `wechataibot/internal/app/llm_loader.go`

## Validation Expectations

- `go test ./...`
- 集成测试前提：系统可找到 `claude` CLI，且存在 `ANTHROPIC_AUTH_TOKEN`

## High-Risk Areas

- `pkg/llm.go`
- `pkg/options.go`
