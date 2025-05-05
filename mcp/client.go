package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"

	"github.com/google/uuid"
)

type Method string

func (m Method) String() string {
	return string(m)
}

const (
	MethodInitialize              Method = "initialize"
	MethodNotificationInitialized Method = "notifications/initialized"
	MethodToolsList               Method = "tools/list"
	MethodToolCall                Method = "tools/call"
)

type JSONRPCRequest struct {
	ID         string         `json:"id,omitempty"`
	RPCVersion string         `json:"jsonrpc"`
	Method     string         `json:"method"`
	Params     map[string]any `json:"params,omitempty"`
}

type JSONRPCResult struct {
	ID         string         `json:"id,omitempty"`
	RPCVersion string         `json:"jsonrpc"`
	Result     map[string]any `json:"result,omitempty"`
	Error      *JSONRPCError  `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type Client struct {
	ServerConfig MCPServerConfig
	Cmd          *exec.Cmd
	Stdin        io.WriteCloser
	Stdout       io.ReadCloser
	Stderr       io.ReadCloser

	// https://modelcontextprotocol.io/docs/concepts/architecture#1-initialization
	initialized bool
}

type Tool struct {
	Name        string
	Description string
	InputSchema ToolInputSchema
}

type ToolInputSchema struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties,omitempty"`
	Required   []string       `json:"required,omitempty"`
}

func NewClient(serverConfig MCPServerConfig) *Client {
	return &Client{
		ServerConfig: serverConfig,
	}
}

func (c *Client) Connect(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, c.ServerConfig.Command, c.ServerConfig.Args...)
	if c.ServerConfig.Env != nil {
		env := os.Environ()
		for k, v := range c.ServerConfig.Env {
			env = append(env, k+"="+v)
		}
		cmd.Env = env
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	c.Cmd = cmd
	c.Stdin = stdin
	c.Stdout = stdout
	c.Stderr = stderr

	return nil
}

func (c *Client) Close(ctx context.Context) error {
	if err := c.Stdin.Close(); err != nil {
		return err
	}

	if err := c.Stdout.Close(); err != nil {
		return err
	}

	if err := c.Stderr.Close(); err != nil {
		return err
	}

	if err := c.Cmd.Process.Kill(); err != nil {
		return err
	}

	return nil
}

// https://modelcontextprotocol.io/docs/concepts/architecture#connection-lifecycle
func (c *Client) Initialize(ctx context.Context) error {
	request := JSONRPCRequest{
		ID:         uuid.New().String(),
		RPCVersion: "2.0",
		Method:     MethodInitialize.String(),
		Params: map[string]any{
			"protocolVersion": "2025-03-26",
			"clientInfo": map[string]any{
				"name":    "scratch-mcp-client",
				"version": "0.1.0",
			},
			"capabilities": map[string]any{},
		},
	}

	result, err := c.sendRequest(ctx, request)
	if err != nil {
		return err
	}

	if result.Error != nil {
		return errors.New("サーバーからエラーが返されました: " + result.Error.Message)
	}

	notificationInitializedRequest := JSONRPCRequest{
		RPCVersion: "2.0",
		Method:     MethodNotificationInitialized.String(),
	}

	if err := c.sendNotification(ctx, notificationInitializedRequest); err != nil {
		return err
	}

	c.initialized = true

	return nil
}

func (c *Client) sendRequest(ctx context.Context, request JSONRPCRequest) (*JSONRPCResult, error) {
	if c.Stdin == nil {
		return nil, errors.New("標準入力が接続されていません")
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	requestJSON = append(requestJSON, '\n')
	if _, err := c.Stdin.Write(requestJSON); err != nil {
		return nil, err
	}

	response := JSONRPCResult{}
	if err := json.NewDecoder(c.Stdout).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) sendNotification(ctx context.Context, request JSONRPCRequest) error {
	if c.Stdin == nil {
		return errors.New("標準入力が接続されていません")
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return err
	}

	requestJSON = append(requestJSON, '\n')
	if _, err := c.Stdin.Write(requestJSON); err != nil {
		return err
	}

	return nil
}

func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	request := JSONRPCRequest{
		ID:         uuid.New().String(),
		RPCVersion: "2.0",
		Method:     MethodToolsList.String(),
		Params:     map[string]any{},
	}

	result, err := c.sendRequest(ctx, request)
	if err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, errors.New("サーバーからエラーが返されました: " + result.Error.Message)
	}

	tools := []Tool{}
	for _, tool := range result.Result["tools"].([]any) {
		toolMap := tool.(map[string]any)
		inputSchema := toolMap["inputSchema"].(map[string]any)

		required := []string{}
		for _, requiredItem := range inputSchema["required"].([]any) {
			required = append(required, requiredItem.(string))
		}

		tools = append(tools, Tool{
			Name:        toolMap["name"].(string),
			Description: toolMap["description"].(string),
			InputSchema: ToolInputSchema{
				Type:       inputSchema["type"].(string),
				Properties: inputSchema["properties"].(map[string]any),
				Required:   required,
			},
		})
	}

	return tools, nil
}

func (c *Client) CallTool(ctx context.Context, name string, params map[string]any) (*JSONRPCResult, error) {
	request := JSONRPCRequest{
		ID:         uuid.New().String(),
		RPCVersion: "2.0",
		Method:     MethodToolCall.String(),
		Params: map[string]any{
			"name":      name,
			"arguments": params,
		},
	}

	result, err := c.sendRequest(ctx, request)
	if err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, errors.New("サーバーからエラーが返されました: " + result.Error.Message)
	}

	return result, nil
}
