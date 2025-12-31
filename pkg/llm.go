package claudecode

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tmc/langchaingo/llms"
)

// LLM wraps the Claude Code CLI and implements llms.Model.
type LLM struct {
	cliPath string
	opts    Options
}

var (
	// ErrCLINotFound is returned when the Claude Code CLI cannot be found.
	ErrCLINotFound = errors.New("claude cli not found")
	// ErrEmptyPrompt is returned when the prompt is empty after formatting.
	ErrEmptyPrompt = errors.New("prompt is empty")
)

// New constructs a Claude Code LLM client.
// 参数：opts 为可选配置项。
// 返回：*LLM 与错误。
func New(opts ...Option) (*LLM, error) {
	options := defaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	cliPath := strings.TrimSpace(options.CLIPath)
	if cliPath == "" {
		// 默认优先使用 ~/.local/bin/claude，再回退到 PATH 搜索。
		if home, err := os.UserHomeDir(); err == nil {
			defaultPath := filepath.Join(home, ".local", "bin", "claude")
			if _, err := exec.LookPath(defaultPath); err == nil {
				cliPath = defaultPath
			}
		}
		if cliPath == "" {
			path, err := exec.LookPath("claude")
			if err != nil {
				return nil, fmt.Errorf("%w: %v", ErrCLINotFound, err)
			}
			cliPath = path
		}
	}

	if options.PermissionMode == "" {
		options.PermissionMode = defaultPermissionMode
	}
	if options.MaxBufferSize <= 0 {
		options.MaxBufferSize = defaultMaxBufferSize
	}
	if options.Env == nil {
		options.Env = map[string]string{}
	}
	if options.ExtraArgs == nil {
		options.ExtraArgs = map[string]string{}
	}

	return &LLM{
		cliPath: cliPath,
		opts:    options,
	}, nil
}

// Call implements llms.Model.Call by delegating to GenerateFromSinglePrompt.
// 参数：ctx 为上下文，prompt 为输入文本，options 为调用参数。
// 返回：模型响应文本与错误。
func (l *LLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return llms.GenerateFromSinglePrompt(ctx, l, prompt, options...)
}

// GenerateContent implements llms.Model.GenerateContent.
// 参数：ctx 为上下文，messages 为对话消息，options 为调用参数。
// 返回：统一的 ContentResponse 与错误。
func (l *LLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) { //nolint:lll
	if l == nil {
		return nil, errors.New("claude code: nil receiver")
	}

	// 解析调用参数，汇总到统一的 CallOptions。
	callOpts := llms.CallOptions{}
	for _, opt := range options {
		opt(&callOpts)
	}

	// 拆分 system 消息与普通消息，避免混入非 system 内容。
	systemFromMessages, nonSystem, err := splitSystemMessages(messages)
	if err != nil {
		return nil, err
	}

	// 合并系统提示词并构建最终 prompt。
	systemPrompt := mergeSystemPrompt(l.opts.SystemPrompt, systemFromMessages)
	prompt, err := buildPrompt(nonSystem)
	if err != nil {
		return nil, err
	}
	// 保障 prompt 非空，避免无效调用。
	if strings.TrimSpace(prompt) == "" {
		return nil, ErrEmptyPrompt
	}

	// 构建 Claude CLI 命令并注入运行环境。
	cmd := l.buildCommand(ctx, prompt, systemPrompt)
	cmd.Env = mergeEnv(os.Environ(), l.opts.Env)
	if l.opts.Cwd != "" {
		cmd.Dir = l.opts.Cwd
	}

	// 建立 stdout/stderr 管道，便于流式读取与错误收集。
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("claude code: stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("claude code: stderr pipe: %w", err)
	}

	// 启动 CLI 子进程。
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("claude code: start cli: %w", err)
	}

	// 异步收集 stderr，防止阻塞主流程。
	var stderrBuf bytes.Buffer
	stderrDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(&stderrBuf, stderr)
		close(stderrDone)
	}()

	// 读取流式输出并捕获生成信息。
	responseText, genInfo, streamErr := l.readStream(ctx, stdout, callOpts.StreamingFunc)
	if streamErr != nil {
		// 出错时强制终止子进程并等待 stderr 收集完成。
		_ = cmd.Process.Kill()
		<-stderrDone
		return nil, streamErr
	}

	// 等待子进程结束并处理可能的 CLI 失败信息。
	if err := cmd.Wait(); err != nil {
		<-stderrDone
		errText := strings.TrimSpace(stderrBuf.String())
		if errText != "" {
			return nil, fmt.Errorf("claude code: cli failed: %w: %s", err, errText)
		}
		return nil, fmt.Errorf("claude code: cli failed: %w", err)
	}
	<-stderrDone

	// 封装为统一的 ContentResponse 返回。
	choice := &llms.ContentChoice{
		Content:        responseText,
		GenerationInfo: genInfo,
	}
	return &llms.ContentResponse{Choices: []*llms.ContentChoice{choice}}, nil
}

// buildCommand builds the CLI command arguments for a single prompt.
// 参数：prompt 为用户输入，systemPrompt 为系统提示词。
// 返回：exec.Cmd。
func (l *LLM) buildCommand(ctx context.Context, prompt string, systemPrompt string) *exec.Cmd {
	args := []string{"--output-format", "stream-json", "--verbose"}

	if systemPrompt != "" {
		args = append(args, "--system-prompt", systemPrompt)
	}
	if len(l.opts.Tools) > 0 {
		args = append(args, "--tools", strings.Join(l.opts.Tools, ","))
	}
	if len(l.opts.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(l.opts.AllowedTools, ","))
	}
	if len(l.opts.DisallowedTools) > 0 {
		args = append(args, "--disallowedTools", strings.Join(l.opts.DisallowedTools, ","))
	}
	if l.opts.Model != "" {
		args = append(args, "--model", l.opts.Model)
	}
	if l.opts.PermissionMode != "" {
		args = append(args, "--permission-mode", l.opts.PermissionMode)
	}

	// Append extra args in stable order for reproducibility.
	if len(l.opts.ExtraArgs) > 0 {
		keys := make([]string, 0, len(l.opts.ExtraArgs))
		for k := range l.opts.ExtraArgs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			val := l.opts.ExtraArgs[key]
			if val == "" {
				args = append(args, "--"+key)
				continue
			}
			args = append(args, "--"+key, val)
		}
	}

	// Use --print with delimiter to avoid prompt being parsed as flags.
	args = append(args, "--print", "--", prompt)

	// 命令样式示例：claude --output-format stream-json --verbose ... --print -- <prompt>
	// 注意：此处会完整输出 prompt，便于排查命令拼装是否正确。
	log.Printf("claude command: %s", strings.Join(append([]string{l.cliPath}, args...), " "))

	return exec.CommandContext(ctx, l.cliPath, args...)
}

// readStream parses stream-json output and returns the aggregated response.
// 参数：ctx 为上下文，stdout 为 CLI 标准输出，streamingFunc 为流式回调。
// 返回：拼接后的文本、生成信息与错误。
func (l *LLM) readStream(ctx context.Context, stdout io.Reader, streamingFunc func(context.Context, []byte) error) (string, map[string]any, error) { //nolint:lll
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), l.opts.MaxBufferSize)

	var builder strings.Builder
	var generationInfo map[string]any

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			return builder.String(), generationInfo, fmt.Errorf("claude code: parse json: %w", err)
		}

		msgType, _ := payload["type"].(string)

		// 记录每行完整 stream-json，包含 msgType，便于排查输出内容。
		logType := msgType
		if logType == "" {
			logType = "<empty>"
		}
		if pretty, err := json.MarshalIndent(payload, "", "  "); err == nil {
			log.Printf("\nclaude code: stream-json (msgType=%s):\n%s\n", logType, string(pretty))
		} else {
			log.Printf("\nclaude code: stream-json (msgType=%s): %s\n", logType, line)
		}

		switch msgType {
		case "assistant":
			texts, err := extractAssistantTexts(payload)
			if err != nil {
				return builder.String(), generationInfo, err
			}
			for _, text := range texts {
				if streamingFunc != nil {
					if err := streamingFunc(ctx, []byte(text)); err != nil {
						return builder.String(), generationInfo, err
					}
				}
				builder.WriteString(text)
			}
		case "result":
			generationInfo = mergeResultInfo(generationInfo, payload)
		case "":
			return builder.String(), generationInfo, fmt.Errorf("claude code: cli error: %v", payload)
		default:
			// Ignore other message types (system, stream_event, etc.).
		}
	}
	if err := scanner.Err(); err != nil {
		return builder.String(), generationInfo, fmt.Errorf("claude code: read stdout: %w", err)
	}

	return builder.String(), generationInfo, nil
}

// splitSystemMessages extracts system messages and returns remaining messages.
// 参数：messages 为原始消息。
// 返回：系统提示文本、非系统消息与错误。
func splitSystemMessages(messages []llms.MessageContent) (string, []llms.MessageContent, error) {
	var systemParts []string
	others := make([]llms.MessageContent, 0, len(messages))

	for _, msg := range messages {
		if msg.Role == llms.ChatMessageTypeSystem {
			text, err := messageToText(msg)
			if err != nil {
				return "", nil, err
			}
			if strings.TrimSpace(text) != "" {
				systemParts = append(systemParts, text)
			}
			continue
		}
		others = append(others, msg)
	}

	return strings.Join(systemParts, "\n\n"), others, nil
}

// mergeSystemPrompt merges the configured system prompt with message system text.
// 参数：base 为配置系统提示，extra 为消息系统提示。
// 返回：合并后的系统提示。
func mergeSystemPrompt(base, extra string) string {
	base = strings.TrimSpace(base)
	extra = strings.TrimSpace(extra)
	if base == "" {
		return extra
	}
	if extra == "" {
		return base
	}
	return base + "\n\n" + extra
}

// buildPrompt formats messages into a single prompt string.
// 参数：messages 为非系统消息。
// 返回：拼接后的 prompt 与错误。
func buildPrompt(messages []llms.MessageContent) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	parts := make([]string, 0, len(messages))
	for _, msg := range messages {
		text, err := messageToText(msg)
		if err != nil {
			return "", err
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}

		prefix := rolePrefix(msg.Role)
		if prefix != "" {
			parts = append(parts, fmt.Sprintf("%s: %s", prefix, text))
			continue
		}
		parts = append(parts, text)
	}

	return strings.Join(parts, "\n\n"), nil
}

// messageToText converts MessageContent to plain text.
// 参数：msg 为消息内容。
// 返回：文本内容与错误。
func messageToText(msg llms.MessageContent) (string, error) {
	if len(msg.Parts) == 0 {
		return "", nil
	}

	var builder strings.Builder
	for i, part := range msg.Parts {
		if i > 0 {
			builder.WriteString("\n")
		}

		switch p := part.(type) {
		case llms.TextContent:
			builder.WriteString(p.Text)
		case llms.ToolCall:
			if p.FunctionCall != nil {
				builder.WriteString(fmt.Sprintf("[ToolCall] %s %s", p.FunctionCall.Name, p.FunctionCall.Arguments))
				continue
			}
			builder.WriteString("[ToolCall]")
		case llms.ToolCallResponse:
			name := strings.TrimSpace(p.Name)
			if name == "" {
				builder.WriteString(fmt.Sprintf("[ToolResult] %s", p.Content))
				continue
			}
			builder.WriteString(fmt.Sprintf("[ToolResult:%s] %s", name, p.Content))
		default:
			return "", fmt.Errorf("claude code: unsupported content part: %T", part)
		}
	}

	return builder.String(), nil
}

// rolePrefix maps chat roles to a readable prefix.
// 参数：role 为消息角色。
// 返回：前缀字符串。
func rolePrefix(role llms.ChatMessageType) string {
	switch role {
	case llms.ChatMessageTypeHuman:
		return "User"
	case llms.ChatMessageTypeAI:
		return "Assistant"
	case llms.ChatMessageTypeFunction:
		return "Function"
	case llms.ChatMessageTypeTool:
		return "Tool"
	case llms.ChatMessageTypeGeneric:
		return "User"
	case llms.ChatMessageTypeSystem:
		return "System"
	default:
		return ""
	}
}

// extractAssistantTexts extracts text blocks from assistant messages.
// 参数：payload 为 CLI JSON 行。
// 返回：文本块与错误。
func extractAssistantTexts(payload map[string]any) ([]string, error) {
	message, ok := payload["message"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("claude code: assistant message missing 'message'")
	}

	content, ok := message["content"]
	if !ok {
		return nil, fmt.Errorf("claude code: assistant message missing 'content'")
	}

	switch blocks := content.(type) {
	case []any:
		texts := make([]string, 0, len(blocks))
		for _, block := range blocks {
			blockMap, ok := block.(map[string]any)
			if !ok {
				continue
			}
			blockType, _ := blockMap["type"].(string)
			if blockType != "text" {
				continue
			}
			text, _ := blockMap["text"].(string)
			if text != "" {
				texts = append(texts, text)
			}
		}
		return texts, nil
	case string:
		if blocks == "" {
			return nil, nil
		}
		return []string{blocks}, nil
	default:
		return nil, fmt.Errorf("claude code: unsupported assistant content type: %T", content)
	}
}

// mergeResultInfo extracts useful fields from result messages.
// 参数：existing 为已有 GenerationInfo，payload 为 result 消息。
// 返回：合并后的 GenerationInfo。
func mergeResultInfo(existing map[string]any, payload map[string]any) map[string]any {
	if existing == nil {
		existing = make(map[string]any)
	}

	if v, ok := payload["total_cost_usd"]; ok {
		existing["TotalCostUSD"] = v
	}
	if v, ok := payload["usage"]; ok {
		existing["Usage"] = v
	}
	if v, ok := payload["result"]; ok {
		existing["Result"] = v
	}
	if v, ok := payload["structured_output"]; ok {
		existing["StructuredOutput"] = v
	}

	return existing
}

// mergeEnv merges base environment with overrides.
// 参数：base 为默认环境，overrides 为覆盖值。
// 返回：合并后的环境变量切片。
func mergeEnv(base []string, overrides map[string]string) []string {
	if len(overrides) == 0 {
		return base
	}
	merged := make([]string, 0, len(base)+len(overrides))
	merged = append(merged, base...)
	for k, v := range overrides {
		merged = append(merged, fmt.Sprintf("%s=%s", k, v))
	}
	return merged
}
