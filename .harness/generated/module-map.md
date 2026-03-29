# Generated Module Map

| Path | Kind | Responsibility |
| --- | --- | --- |
| `pkg/llm.go` | runtime entry | CLI 发现、命令拼装、流式读取、`llms.Model` 实现 |
| `pkg/options.go` | contract | 选项体系、输出模式、工具事件、thinking tag、session 参数 |
| `pkg/llm_test.go` | unit test | stream-json 解析、段落拼接、thinking tag 输出语义 |
| `pkg/llm_glm_test.go` | integration test | GLM 兼容接口验证 |
| `pkg/llm_ds_test.go` | integration test | DeepSeek 兼容接口验证 |
