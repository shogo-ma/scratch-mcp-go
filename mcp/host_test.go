package mcp

import (
	"testing"
)

func TestConvertToAnthropicTools(t *testing.T) {
	tools := []Tool{
		{
			Name:        "echo",
			Description: "エコーツール",
			InputSchema: ToolInputSchema{
				Properties: map[string]any{
					"message": map[string]any{"type": "string"},
				},
			},
		},
	}

	anthropicTools := convertToAnthropicTools(tools)
	if len(anthropicTools) != 1 {
		t.Fatalf("expected 1 anthropic tool, got %d", len(anthropicTools))
	}

	tool := anthropicTools[0]
	if tool.OfTool == nil {
		t.Fatal("OfTool is nil")
	}
	if tool.OfTool.Name != "echo" {
		t.Errorf("expected tool name 'echo', got '%s'", tool.OfTool.Name)
	}
	if tool.OfTool.Description.Value != "エコーツール" {
		t.Errorf("expected description 'エコーツール', got '%v'", tool.OfTool.Description.Value)
	}
	props, ok := tool.OfTool.InputSchema.Properties.(map[string]any)
	if !ok {
		t.Fatalf("expected Properties to be map[string]any")
	}
	if prop, ok := props["message"].(map[string]any); !ok || prop["type"] != "string" {
		t.Errorf("expected property 'message' of type 'string'")
	}
}

func TestConvertToAnthropicMessages(t *testing.T) {
	messages := []Message{
		{
			Role:    RoleUser,
			Content: &Content{Content: "こんにちは"},
		},
		{
			Role:    RoleAssistant,
			Content: &Content{Content: "やあ！"},
		},
	}

	anthropicMessages := convertToAnthropicMessages(messages)
	if len(anthropicMessages) != 2 {
		t.Fatalf("expected 2 anthropic messages, got %d", len(anthropicMessages))
	}
	if string(anthropicMessages[0].Role) != string(RoleUser) {
		t.Errorf("expected role 'user', got '%s'", anthropicMessages[0].Role)
	}
	if anthropicMessages[0].Content[0].OfRequestTextBlock == nil || anthropicMessages[0].Content[0].OfRequestTextBlock.Text != "こんにちは" {
		t.Errorf("expected content 'こんにちは', got '%v'", anthropicMessages[0].Content[0].OfRequestTextBlock)
	}
	if string(anthropicMessages[1].Role) != string(RoleAssistant) {
		t.Errorf("expected role 'assistant', got '%s'", anthropicMessages[1].Role)
	}
	if anthropicMessages[1].Content[0].OfRequestTextBlock == nil || anthropicMessages[1].Content[0].OfRequestTextBlock.Text != "やあ！" {
		t.Errorf("expected content 'やあ！', got '%v'", anthropicMessages[1].Content[0].OfRequestTextBlock)
	}
}
