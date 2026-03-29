# Conventions

## CLI Semantics

- 默认优先查找 `~/.local/bin/claude`，再回退到 PATH
- 默认 permission mode 是 `bypassPermissions`
- `ExtraArgs` 按 key 排序后拼接，保证命令参数顺序稳定

## Option Layer

- `Option` 只负责变异 `Options`
- 默认 `OutputMode` 是 `OutputModeText`，不会把工具事件写进最终输出
- `ThinkingTags` 默认为关闭，开启后才把 Claude 的 thinking block 渲染成 `<think>...</think>`
- session 参数通过 `WithSessionID`、`WithResume`、`WithForkSession`、`WithNoSessionPersistence` 组合表达
- `WithResume(true)` + `WithSessionID(id)` 映射为 `--resume <id>`；未开启 `Resume` 时才会使用 `--session-id <id>`
- `WithForkSession` 只有在恢复语义下才有意义；`WithNoSessionPersistence` 只对 `--print` 路径有效

## Boundary Discipline

- 适配层不应硬编码上层产品的 skill 路径或业务命令
- 模型与环境默认值可以通过 `WithEnv` 覆盖
- README 中出现的集成测试环境变量写法如果与代码冲突，应以 `pkg/*_test.go` 和 runbook 为准
