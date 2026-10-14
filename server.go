package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

const (
	MCPSpecVersion = "2026-07-28"
	JSONRPCVersion = "2.0"
)

// Server manages the MCP server state and communication
type Server struct {
	Name        string
	Version     string
	Description string
	Tools       []Tool
}

// NewServer creates a new MCP server instance
func NewServer(name, version, description string) *Server {
	return &Server{
		Name:        name,
		Version:     version,
		Description: description,
		Tools:       []Tool{},
	}
}

// AddTool registers a new tool with the server
func (s *Server) AddTool(t Tool) {
	s.Tools = append(s.Tools, t)
}

// Serve starts the JSON-RPC loop over the provided reader and writer
func (s *Server) Serve(stdin io.Reader, stdout io.Writer) {
	scanner := bufio.NewScanner(stdin)
	for scanner.Scan() {
		var req RPCRequest
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			s.respond(stdout, nil, nil, &RPCError{
				Code:    -32700,
				Message: "Parse error",
			})
			continue
		}

		switch req.Method {
		case "server/discover":
			discoverRes := DiscoverResult{
				MCPResult: MCPResult{
					ResultType: "complete",
				},
				SupportedVersions: []string{MCPSpecVersion},
				Capabilities: map[string]interface{}{
					"tools": map[string]interface{}{},
				},
				Meta: map[string]interface{}{
					"io.modelcontextprotocol/serverInfo": map[string]string{
						"name":    s.Name,
						"version": s.Version,
					},
				},
				Instructions: s.Description,
			}
			s.respond(stdout, req.ID, discoverRes, nil)

		case "tools/list":
			var list []Tool
			for _, t := range s.Tools {
				list = append(list, Tool{
					Name:         t.Name,
					Title:        t.Title,
					Description:  t.Description,
					InputSchema:  t.InputSchema,
					OutputSchema: t.OutputSchema,
				})
			}
			s.respond(stdout, req.ID, map[string]interface{}{
				"tools": list,
			}, nil)

		case "tools/call":
			result, rpcErr := s.handleToolCall(req.Params)
			s.respond(stdout, req.ID, result, rpcErr)

		default:
			s.respond(stdout, req.ID, nil, &RPCError{
				Code:    -32601,
				Message: "Method not found",
			})
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
	}
}

func (s *Server) handleToolCall(params json.RawMessage) (any, *RPCError) {
	var call ToolCallParams
	if err := json.Unmarshal(params, &call); err != nil {
		return nil, &RPCError{Code: -32602, Message: "Invalid params"}
	}

	for _, tool := range s.Tools {
		if tool.Name == call.Name {
			return tool.Handler(call.Arguments)
		}
	}

	return nil, &RPCError{Code: -32601, Message: "Tool not found"}
}

func (s *Server) respond(w io.Writer, id any, result any, rpcErr *RPCError) {
	resp := RPCResponse{
		JSONRPC: JSONRPCVersion,
		ID:      id,
	}

	if rpcErr != nil {
		resp.Error = rpcErr
	} else if result != nil {
		resBytes, err := json.Marshal(result)
		if err != nil {
			resp.Error = &RPCError{Code: -32603, Message: "Internal JSON marshal error"}
		} else {
			resp.Result = resBytes
		}
	}

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(resp); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// Types

type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type MCPResult struct {
	ResultType string `json:"resultType"`
}

type DiscoverResult struct {
	MCPResult
	SupportedVersions []string               `json:"supportedVersions"`
	Capabilities      map[string]interface{} `json:"capabilities"`
	Meta              map[string]interface{} `json:"_meta"`
	Instructions      string                 `json:"instructions"`
}

type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type ToolResult struct {
	MCPResult
	Content           []ToolContent `json:"content"`
	StructuredContent any           `json:"structuredContent,omitempty"`
}

type Tool struct {
	Name         string                                                `json:"name"`
	Title        string                                                `json:"title,omitempty"`
	Description  string                                                `json:"description"`
	InputSchema  *ToolSchema                                           `json:"inputSchema"`
	OutputSchema *ToolSchema                                           `json:"outputSchema,omitempty"`
	Handler      func(map[string]interface{}) (*ToolResult, *RPCError) `json:"-"`
}

type ToolSchema struct {
	Type       string          `json:"type"`
	Properties json.RawMessage `json:"properties,omitempty"`
	Required   []string        `json:"required,omitempty"`
}

type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type Param struct {
	Name        string
	Type        string
	Description string
	Required    bool
}

// Options

type ToolOption func(*Tool)

func WithTitle(title string) ToolOption {
	return func(t *Tool) {
		t.Title = title
	}
}

func WithDescription(desc string) ToolOption {
	return func(t *Tool) {
		t.Description = desc
	}
}

func WithParams(params ...Param) ToolOption {
	return func(t *Tool) {
		props := make(map[string]interface{})
		var required []string

		for _, p := range params {
			field := map[string]string{"type": p.Type}
			if p.Description != "" {
				field["description"] = p.Description
			}
			props[p.Name] = field
			if p.Required {
				required = append(required, p.Name)
			}
		}

		bytes, err := json.Marshal(props)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error marshaling params for tool %s: %v\n", t.Name, err)
			return
		}
		t.InputSchema = &ToolSchema{
			Type:       "object",
			Properties: bytes,
			Required:   required,
		}
	}
}

func WithOutputSchema(params ...Param) ToolOption {
	return func(t *Tool) {
		props := make(map[string]interface{})
		var required []string

		for _, p := range params {
			field := map[string]string{"type": p.Type}
			if p.Description != "" {
				field["description"] = p.Description
			}
			props[p.Name] = field
			if p.Required {
				required = append(required, p.Name)
			}
		}

		bytes, err := json.Marshal(props)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error marshaling output schema for tool %s: %v\n", t.Name, err)
			return
		}
		t.OutputSchema = &ToolSchema{
			Type:       "object",
			Properties: bytes,
			Required:   required,
		}
	}
}

func NewTool(name string, handler func(map[string]interface{}) (*ToolResult, *RPCError), opts ...ToolOption) Tool {
	t := Tool{
		Name:    name,
		Handler: handler,
	}
	for _, opt := range opts {
		opt(&t)
	}
	return t
}
