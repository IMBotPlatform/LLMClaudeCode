package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/IMBotPlatform/LLMClaudeCode/claudecode"
	"github.com/tmc/langchaingo/llms"
)

// main 作为容器内的简单运行入口。
// 参数：通过命令行传入 prompt 与可选配置。
// 返回：标准输出模型响应，失败时退出非零。
func main() {
	var (
		model          = flag.String("model", "", "Claude model name")
		systemPrompt   = flag.String("system", "", "System prompt")
		cliPath        = flag.String("cli", "", "Path to Claude Code CLI")
		permissionMode = flag.String("permission-mode", "bypassPermissions", "Permission mode")
		cwd            = flag.String("cwd", "", "Working directory")
	)
	flag.Parse()

	prompt := strings.TrimSpace(strings.Join(flag.Args(), " "))
	if prompt == "" {
		stdin, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("read stdin: %v", err)
		}
		prompt = strings.TrimSpace(string(stdin))
	}
	if prompt == "" {
		log.Fatal("prompt is required")
	}

	opts := []claudecode.Option{}
	if *model != "" {
		opts = append(opts, claudecode.WithModel(*model))
	}
	if *systemPrompt != "" {
		opts = append(opts, claudecode.WithSystemPrompt(*systemPrompt))
	}
	if *cliPath != "" {
		opts = append(opts, claudecode.WithCLIPath(*cliPath))
	}
	if *permissionMode != "" {
		opts = append(opts, claudecode.WithPermissionMode(*permissionMode))
	}
	if *cwd != "" {
		opts = append(opts, claudecode.WithCwd(*cwd))
	}

	llm, err := claudecode.New(opts...)
	if err != nil {
		log.Fatalf("init claudecode: %v", err)
	}

	ctx := context.Background()
	resp, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt)
	if err != nil {
		log.Fatalf("claudecode error: %v", err)
	}

	fmt.Print(resp)
}
