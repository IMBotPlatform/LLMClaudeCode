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
	"time"

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

	// Session management - 互斥处理：--resume 和 --session-id 不能同时使用
	if l.opts.Resume {
		// 恢复会话：只用 --resume <id>
		if l.opts.SessionID != "" {
			args = append(args, "--resume", l.opts.SessionID)
		} else {
			args = append(args, "--resume")
		}
	} else if l.opts.SessionID != "" {
		// 新会话：只用 --session-id <id>
		args = append(args, "--session-id", l.opts.SessionID)
	}
	if l.opts.ForkSession {
		args = append(args, "--fork-session")
	}

	if l.opts.NoSessionPersistence {
		args = append(args, "--no-session-persistence")
	}

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
	var pendingThinking []string
	emittedVisibleText := false

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
			// 处理 assistant 消息中的有序内容块（text / thinking / tool_use）
			blocks, err := extractAssistantContent(payload)
			if err != nil {
				return builder.String(), generationInfo, err
			}
			for _, block := range blocks {
				switch block.Kind {
				case assistantContentText:
					chunk := block.Text
					if !emittedVisibleText && l.opts.ThinkingTags {
						if thinkChunk := formatThinkingBlock(joinThinkingBlocks(pendingThinking)); thinkChunk != "" {
							chunk = thinkChunk + "\n\n" + block.Text
							pendingThinking = nil
						}
					} else if shouldInsertAssistantParagraphBreak(builder.String(), block.Text) {
						chunk = "\n\n" + block.Text
					}
					if streamingFunc != nil {
						if err := streamingFunc(ctx, []byte(chunk)); err != nil {
							return builder.String(), generationInfo, err
						}
					}
					builder.WriteString(chunk)
					emittedVisibleText = emittedVisibleText || strings.TrimSpace(block.Text) != ""
				case assistantContentThinking:
					if !l.opts.ThinkingTags {
						continue
					}
					if emittedVisibleText {
						continue
					}
					pendingThinking = append(pendingThinking, block.Text)
				case assistantContentToolUse:
					tu := block.ToolUse
					l.handleToolEvent(ToolEvent{
						Type:      ToolEventUse,
						ToolName:  tu.Name,
						ToolID:    tu.ID,
						Input:     tu.Input,
						Timestamp: time.Now(),
					}, &builder, streamingFunc, ctx)
				}
			}
		case "tool_result":
			// 处理工具执行结果消息
			l.handleToolEvent(ToolEvent{
				Type:      ToolEventResult,
				ToolID:    getStringField(payload, "tool_use_id"),
				Output:    getStringField(payload, "content"),
				Timestamp: time.Now(),
			}, &builder, streamingFunc, ctx)
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

// shouldInsertAssistantParagraphBreak 判断新的 assistant 文本块前是否需要补段落分隔。
func shouldInsertAssistantParagraphBreak(existing, next string) bool {
	if strings.TrimSpace(existing) == "" || strings.TrimSpace(next) == "" {
		return false
	}
	if strings.HasPrefix(next, "\n") {
		return false
	}
	if strings.HasSuffix(existing, "\n") {
		return false
	}
	return true
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

// toolUseInfo 工具调用信息。
type toolUseInfo struct {
	ID    string
	Name  string
	Input map[string]any
}

type assistantContentKind int

const (
	assistantContentText assistantContentKind = iota
	assistantContentThinking
	assistantContentToolUse
)

type assistantContentBlock struct {
	Kind    assistantContentKind
	Text    string
	ToolUse toolUseInfo
}

// extractAssistantContent extracts ordered text/thinking/tool_use blocks from assistant messages.
// 参数：payload 为 CLI JSON 行。
// 返回：有序内容块与错误。
func extractAssistantContent(payload map[string]any) ([]assistantContentBlock, error) {
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
		var out []assistantContentBlock
		for _, block := range blocks {
			blockMap, ok := block.(map[string]any)
			if !ok {
				continue
			}
			blockType, _ := blockMap["type"].(string)
			switch blockType {
			case "text":
				text, _ := blockMap["text"].(string)
				if text != "" {
					out = append(out, assistantContentBlock{
						Kind: assistantContentText,
						Text: text,
					})
				}
			case "thinking":
				text, _ := blockMap["thinking"].(string)
				if text != "" {
					out = append(out, assistantContentBlock{
						Kind: assistantContentThinking,
						Text: text,
					})
				}
			case "tool_use":
				tu := toolUseInfo{
					ID:   getStringField(blockMap, "id"),
					Name: getStringField(blockMap, "name"),
				}
				if input, ok := blockMap["input"].(map[string]any); ok {
					tu.Input = input
				}
				out = append(out, assistantContentBlock{
					Kind:    assistantContentToolUse,
					ToolUse: tu,
				})
			}
		}
		return out, nil
	case string:
		if blocks == "" {
			return nil, nil
		}
		return []assistantContentBlock{{
			Kind: assistantContentText,
			Text: blocks,
		}}, nil
	default:
		return nil, fmt.Errorf("claude code: unsupported assistant content type: %T", content)
	}
}

// formatThinkingBlock wraps thinking text in enterprise-wecom compatible think tags.
func formatThinkingBlock(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	return "<think>" + text + "</think>"
}

// joinThinkingBlocks 将多个 thinking block 聚合成一个思考段落。
func joinThinkingBlocks(blocks []string) string {
	parts := make([]string, 0, len(blocks))
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		parts = append(parts, block)
	}
	return strings.Join(parts, "\n\n")
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

// getStringField safely extracts a string field from a map.
// 参数：m 为 map，key 为字段名。
// 返回：字段值，如果不存在或类型不匹配则返回空字符串。
func getStringField(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// handleToolEvent processes tool events based on OutputMode settings.
// 参数：event 为工具事件，builder 为输出构建器，streamingFunc 为流式回调，ctx 为上下文。
func (l *LLM) handleToolEvent(event ToolEvent, builder *strings.Builder, streamingFunc func(context.Context, []byte) error, ctx context.Context) {
	// 始终触发回调（如果已设置）
	if l.opts.ToolEventHook != nil {
		l.opts.ToolEventHook(event)
	}

	// 根据 OutputMode 决定是否追加到输出
	if l.opts.OutputMode == OutputModeText {
		return // 默认模式不输出工具信息
	}

	var summary string
	switch event.Type {
	case ToolEventUse:
		if l.opts.OutputMode == OutputModeFull {
			// 完整模式：输出工具名称和输入参数
			inputJSON, _ := json.MarshalIndent(event.Input, "", "  ")
			summary = fmt.Sprintf("\n🔧 [%s] %s\n%s\n", event.ToolName, event.ToolID, string(inputJSON))
		} else {
			// Verbose 模式：输出工具名称 + 关键参数摘要
			summary = formatToolUseSummary(event.ToolName, event.Input)
		}
	case ToolEventResult:
		if l.opts.OutputMode == OutputModeFull {
			// 完整模式：输出完整结果
			output := event.Output
			if len(output) > 500 {
				output = output[:500] + "... (truncated)"
			}
			summary = fmt.Sprintf("  └─ 📤 %s\n", output)
		}
		// Verbose 模式不输出 result（避免太冗长）
	}

	if summary != "" {
		builder.WriteString(summary)
		if streamingFunc != nil {
			_ = streamingFunc(ctx, []byte(summary))
		}
	}
}

// formatToolUseSummary 为 Verbose 模式格式化工具调用摘要。
// 参数：toolName 为工具名称，input 为输入参数。
// 返回：格式化的摘要字符串。
func formatToolUseSummary(toolName string, input map[string]any) string {
	// 根据工具类型提取关键参数
	var detail string
	switch toolName {
	case "Read", "read_file", "view_file":
		if path, ok := input["file_path"].(string); ok {
			detail = path
		} else if path, ok := input["path"].(string); ok {
			detail = path
		}
	case "Write", "write_file", "create_file":
		if path, ok := input["file_path"].(string); ok {
			detail = path
		} else if path, ok := input["path"].(string); ok {
			detail = path
		}
	case "Bash", "run_command", "execute_command":
		if cmd, ok := input["command"].(string); ok {
			// 截断过长的命令
			if len(cmd) > 80 {
				detail = cmd[:77] + "..."
			} else {
				detail = cmd
			}
		}
	case "TodoWrite", "task", "plan":
		if todos, ok := input["todos"].(string); ok {
			lines := strings.Split(todos, "\n")
			if len(lines) > 0 {
				detail = fmt.Sprintf("%d items", len(lines))
			}
		}
	case "Skill", "use_skill":
		if name, ok := input["skill_name"].(string); ok {
			detail = name
		} else if name, ok := input["name"].(string); ok {
			detail = name
		}
	case "Search", "grep", "find":
		if query, ok := input["query"].(string); ok {
			detail = query
		} else if pattern, ok := input["pattern"].(string); ok {
			detail = pattern
		}
	default:
		// 尝试提取常见字段
		for _, key := range []string{"path", "file", "command", "query", "name", "url"} {
			if v, ok := input[key].(string); ok && v != "" {
				if len(v) > 60 {
					detail = v[:57] + "..."
				} else {
					detail = v
				}
				break
			}
		}
	}

	if detail != "" {
		return fmt.Sprintf("\n🔧 %s: %s\n", toolName, detail)
	}
	return fmt.Sprintf("\n🔧 %s\n", toolName)
}
