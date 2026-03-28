# Glossary

## stream-json

- Definition: Claude Code CLI 的流式 JSON 输出格式，本仓库按行解析它
- Evidence: `pkg/llm.go`

## PermissionMode

- Definition: CLI 工具权限模式，默认 `bypassPermissions`
- Evidence: `pkg/options.go`

## SessionID / Resume

- Definition: 控制会话恢复与续写的配置项
- Evidence: `pkg/options.go`, `pkg/llm.go`

## ToolEventHook

- Definition: 工具调用事件的观察钩子
- Evidence: `pkg/options.go`
