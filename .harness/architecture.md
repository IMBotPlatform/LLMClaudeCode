# Architecture

## Scope

本文件描述 `pkg` 当前唯一公开包的结构与行为。

## System Shape

Observed fact:

- 仓库只有一个主公开包：`pkg`
- `llm.go` 负责 CLI 调用和流式解析
- `options.go` 负责 Option 与 session 相关配置

## Major Modules

- `pkg/llm.go`
  - `LLM`
  - `New`
  - `GenerateContent`
  - CLI 进程创建、stdout/stderr 管理、stream-json 读取
- `pkg/options.go`
  - `Options`
  - `Option`
  - `With*` 系列配置器
  - `OutputMode`、`ToolEvent`、session 配置

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
  -> parse chunks into llms.ContentResponse
```

## High-Risk Areas

- CLI 参数拼装顺序
- stdout/stderr 读取与进程生命周期
- `SessionID` / `Resume` / `ForkSession` 的互斥逻辑

## Evidence

- `pkg/llm.go`
- `pkg/options.go`
- `pkg/llm_glm_test.go`
- `pkg/llm_ds_test.go`
