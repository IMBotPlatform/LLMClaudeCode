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

## Main Flows

- 构造：解析选项、定位 CLI、设置默认值
- 执行：拼接命令行参数、注入环境变量、启动子进程
- 解析：消费 `stream-json` 输出，转换为 `ContentResponse`

## Important Constraints

- `--resume` 与 `--session-id` 的组合语义需要一致
- stderr 需要异步收集，避免阻塞主流程

## Evidence

- `pkg/llm.go`
- `pkg/options.go`
