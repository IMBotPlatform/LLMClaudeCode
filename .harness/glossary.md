# Glossary

## stream-json

- Definition: Claude Code CLI 的流式 JSON 输出格式，本仓库按行解析它
- Evidence: `pkg/llm.go`

## PermissionMode

- Definition: CLI 工具权限模式，默认 `bypassPermissions`
- Evidence: `pkg/options.go`

## OutputMode

- Definition: 控制最终输出里是否附带工具调用摘要或完整工具结果
- Evidence: `pkg/options.go`, `pkg/llm.go`

## SessionID / Resume

- Definition: 控制会话恢复与续写的配置项
- Evidence: `pkg/options.go`, `pkg/llm.go`

## ToolEventHook

- Definition: 工具调用事件的观察钩子
- Evidence: `pkg/options.go`

## ThinkingTags

- Definition: 控制是否把 Claude 的 thinking block 渲染为 `<think>...</think>` 文本
- Evidence: `pkg/options.go`, `pkg/llm.go`, `pkg/llm_test.go`
