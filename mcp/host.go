package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type Host struct {
	AnthropicClient anthropic.Client
	ClientMap       map[string]*Client
}

func NewHost(ctx context.Context, apiKey, configPath string) (*Host, error) {
	anthropicClient := anthropic.NewClient(option.WithAPIKey(apiKey))

	mcpConfig, err := LoadMCPConfig(configPath)
	if err != nil {
		return nil, err
	}

	clientMap := make(map[string]*Client)

	// clientとserverは1:1
	for name, server := range mcpConfig.McpServers {
		client := NewClient(server)
		if err := client.Connect(ctx); err != nil {
			return nil, err
		}

		if err := client.Initialize(ctx); err != nil {
			return nil, err
		}

		clientMap[name] = client
	}

	return &Host{
		AnthropicClient: anthropicClient,
		ClientMap:       clientMap,
	}, nil
}

func (h *Host) Start(ctx context.Context) error {
	clientToolMap := make(map[string]string)
	tools := []Tool{}
	for name, client := range h.ClientMap {
		clientTools, err := client.ListTools(ctx)
		if err != nil {
			return fmt.Errorf("ツール一覧の取得に失敗しました: %w", err)
		}

		for _, tool := range clientTools {
			clientToolMap[tool.Name] = name
		}

		tools = append(tools, clientTools...)
	}

	scanner := bufio.NewScanner(os.Stdin)

	messages := []Message{}
	for {
		fmt.Print("prompt: ")
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("標準入力の読み取り中にエラーが発生しました: %w", err)
			}
			break
		}

		messages = append(messages, Message{
			Role: RoleUser,
			Content: &Content{
				Content: scanner.Text(),
			},
		})

		message, err := processToolLLMQuery(ctx, h, tools, messages)
		if err != nil {
			return fmt.Errorf("LLMへの問い合わせに失敗しました: %w", err)
		}

		if len(message.Content) == 0 {
			slog.Warn("LLMの回答がありませんでした")
			continue
		}

		assistantMessageContent := []string{}
		outputText := "" // 最終的な出力
		for _, content := range message.Content {
			if content.Type == "text" {
				assistantMessageContent = append(assistantMessageContent, content.Text)
				outputText += fmt.Sprintf("%s\n", content.Text)
			} else if content.Type == "tool_use" {
				clientName, ok := clientToolMap[content.Name]
				if !ok {
					slog.Warn("ツールが見つかりませんでした", slog.Any("name", content.Name))
					continue
				}

				client := h.ClientMap[clientName]

				if content.Text != "" {
					assistantMessageContent = append(assistantMessageContent, content.Text)
				}

				var toolArgs map[string]any
				if err := json.Unmarshal(content.Input, &toolArgs); err != nil {
					slog.Warn("ツール引数のパースに失敗しました", slog.Any("error", err))
					continue
				}

				slog.Info("ツールを実行します", slog.String("name", content.Name))
				result, err := client.CallTool(ctx, content.Name, toolArgs)
				if err != nil {
					slog.Warn("ツールの実行に失敗しました", slog.Any("error", err))
					continue
				}

				resultBytes, err := json.Marshal(result)
				if err != nil {
					slog.Warn("ツールの実行結果のパースに失敗しました", slog.Any("error", err))
					continue
				}

				messages = append(messages, []Message{
					{
						Role: RoleAssistant,
						Content: &Content{
							Content: strings.Join(assistantMessageContent, "\n"),
						},
					},
					{
						Role: RoleAssistant,
						ToolUse: &ToolUse{
							Type:  "tool_use",
							Name:  content.Name,
							ID:    result.ID,
							Input: toolArgs,
						},
					},
					{
						Role: RoleUser,
						ToolResult: &ToolResultContent{
							Type:      "tool_result",
							ToolUseID: result.ID,
							Content:   string(resultBytes),
						},
					},
				}...)

				message, err := processToolLLMQuery(ctx, h, tools, messages)
				if err != nil {
					return err
				}

				outputText += fmt.Sprintf("%s\n", message.Content[0].Text)
			}
		}

		fmt.Println(outputText)
	}

	return nil
}

func convertToAnthropicTools(tools []Tool) []anthropic.ToolUnionParam {
	anthropicTools := []anthropic.ToolUnionParam{}
	for _, tool := range tools {
		anthropicTools = append(anthropicTools, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        tool.Name,
				Description: anthropic.String(tool.Description),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: tool.InputSchema.Properties,
				},
			},
		})
	}

	return anthropicTools
}

func convertToAnthropicMessages(messages []Message) []anthropic.MessageParam {
	anthropicMessages := []anthropic.MessageParam{}
	for _, message := range messages {
		if message.Content != nil {
			content := message.Content.Content
			anthropicMessages = append(anthropicMessages, anthropic.MessageParam{
				Role: anthropic.MessageParamRole(message.Role),
				Content: []anthropic.ContentBlockParamUnion{{
					OfRequestTextBlock: &anthropic.TextBlockParam{
						Text: content,
					},
				}},
			})
		} else if message.ToolUse != nil {
			anthropicMessages = append(anthropicMessages, anthropic.MessageParam{
				Role: anthropic.MessageParamRole(message.Role),
				Content: []anthropic.ContentBlockParamUnion{{
					OfRequestToolUseBlock: &anthropic.ToolUseBlockParam{
						Type:  "tool_use",
						ID:    message.ToolUse.ID,
						Name:  message.ToolUse.Name,
						Input: message.ToolUse.Input,
					},
				}},
			})
		} else if message.ToolResult != nil {
			anthropicMessages = append(anthropicMessages, anthropic.MessageParam{
				Role: anthropic.MessageParamRole(message.Role),
				Content: []anthropic.ContentBlockParamUnion{{
					OfRequestToolResultBlock: &anthropic.ToolResultBlockParam{
						Type:      "tool_result",
						ToolUseID: message.ToolResult.ToolUseID,
						Content: []anthropic.ToolResultBlockParamContentUnion{{
							OfRequestTextBlock: &anthropic.TextBlockParam{
								Text: message.ToolResult.Content,
							},
						}},
					},
				}},
			})
		}
	}

	return anthropicMessages
}

func processToolLLMQuery(
	ctx context.Context,
	host *Host,
	tools []Tool,
	messages []Message,
) (*anthropic.Message, error) {
	anthropicTools := convertToAnthropicTools(tools)
	anthropicMessages := convertToAnthropicMessages(messages)

	message, err := host.AnthropicClient.Messages.New(ctx, anthropic.MessageNewParams{
		MaxTokens: 1024,
		Messages:  anthropicMessages,
		Tools:     anthropicTools,
		Model:     anthropic.ModelClaude3_7SonnetLatest,
	})
	if err != nil {
		return nil, err
	}

	return message, nil
}
