# Architecture

## Scope

本文件描述 `pkg` 当前唯一公开包的结构与行为。

## System Shape

Observed fact:

- 仓库只有一个主公开包：`pkg`
- `llm.go` 负责 prompt 组装、CLI 调用、`stream-json` 解析、thinking/tool 事件格式化
- `options.go` 负责 Option、输出模式、thinking tag、session 相关配置
- `llm_test.go` 负责无外部依赖的流式解析与 thinking tag 单元测试

## Major Modules

- `pkg/llm.go`
  - `LLM`
  - `New`
  - `GenerateContent`
  - CLI 进程创建、stdout/stderr 管理、stream-json 读取
  - thinking block 渲染、tool_use / tool_result 摘要输出
- `pkg/options.go`
  - `Options`
  - `Option`
  - `With*` 系列配置器
  - `OutputMode`、`ToolEvent`、`ThinkingTags`、session 配置
- `pkg/llm_test.go`
  - `shouldInsertAssistantParagraphBreak`
  - `extractAssistantContent`
  - thinking tag 输出语义

## Dependency Directions

- 本仓库依赖 `github.com/tmc/langchaingo/llms`
- `wechataibot` 在 workspace 内直接依赖本仓库

## Key Flow

```text
New(opts...)
  -> resolve claude CLI path
  -> merge options and env
  -> build command args
  -> run claude --output-format stream-json
  -> parse assistant text / thinking / tool_use blocks
  -> optionally emit tool summaries and <think> blocks
  -> fold result payload into llms.ContentResponse
```

## High-Risk Areas

- CLI 参数拼装顺序
- stdout/stderr 读取与进程生命周期
- thinking block 与正文之间的拼接边界
- OutputMode 下的工具事件可见性
- `SessionID` / `Resume` / `ForkSession` 的互斥逻辑

## Evidence

- `pkg/llm.go`
- `pkg/options.go`
- `pkg/llm_test.go`
- `pkg/llm_glm_test.go`
- `pkg/llm_ds_test.go`
