package mcp

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type ContentType string

const (
	ContentTypeText       ContentType = "text"
	ContentTypeToolResult ContentType = "tool_result"
)

type Message struct {
	Role       Role
	Content    *Content
	ToolResult *ToolResultContent
	ToolUse    *ToolUse
}

type Content struct {
	Content string
}

type ToolResultContent struct {
	Type      string
	ToolUseID string
	Content   string
}

type ToolUse struct {
	Type  string
	Name  string
	ID    string
	Input map[string]any
}
