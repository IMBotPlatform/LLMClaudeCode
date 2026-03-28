# Conventions

## CLI Semantics

- 默认优先查找 `~/.local/bin/claude`，再回退到 PATH
- 默认 permission mode 是 `bypassPermissions`

## Option Layer

- `Option` 只负责变异 `Options`
- session 参数通过 `WithSessionID`、`WithResume`、`WithForkSession`、`WithNoSessionPersistence` 组合表达

## Boundary Discipline

- 适配层不应硬编码上层产品的 skill 路径或业务命令
- 模型与环境默认值可以通过 `WithEnv` 覆盖
