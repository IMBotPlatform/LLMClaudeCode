# Domain: Claude CLI Adapter

## Responsibility

将 Claude Code CLI 暴露为 `llms.Model` 实现。

## Non-responsibility

- 不负责产品级 skill 路由
- 不负责 WeCom 平台接入

## Key Concepts

- `LLM`
- `Options`
- `Option`
- `OutputMode`
- `ToolEvent`
- `ThinkingTags`

## Main Flows

- 构造：解析选项、定位 CLI、设置默认值
- 执行：拼接命令行参数、注入环境变量、启动子进程
- 解析：消费 `stream-json` 输出，按顺序处理 `text` / `thinking` / `tool_use`
- 汇总：根据 `OutputMode` 决定是否附加工具摘要，根据 `ThinkingTags` 决定是否输出 `<think>` 块

## Important Constraints

- `--resume` 与 `--session-id` 的组合语义需要一致
- stderr 需要异步收集，避免阻塞主流程
- thinking block 的可见性是下游可观察行为，变更后需要同步检查 `wechataibot`
- `tool_result` 只有在 `OutputModeFull` 下才会写入最终输出

## Evidence

- `pkg/llm.go`
- `pkg/options.go`
- `pkg/llm_test.go`
