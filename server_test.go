package mcp

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestServer_Discover(t *testing.T) {
	// 1. Setup the server
	srv := NewServer("TestServer", "1.0.0", "A test description")

	// 2. Simulate Standard Input (the request)
	// We add a newline \n because bufio.Scanner reads line by line
	input := `{"jsonrpc": "2.0", "id": 1, "method": "server/discover"}` + "\n"
	stdin := bytes.NewBufferString(input)

	// 3. Simulate Standard Output (the response)
	stdout := &bytes.Buffer{}

	// 4. Run the server in a goroutine
	// Because srv.Serve runs an infinite loop (scanner.Scan),
	// it will block the test unless we put it in the background.
	go func() {
		srv.Serve(stdin, stdout)
	}()

	// Give the goroutine a moment to process the input
	time.Sleep(100 * time.Millisecond)

	// 5. Verify the results
	response := stdout.String()

	if response == "" {
		t.Fatal("Server produced no output")
	}

	// Check if the server name we provided is in the JSON response
	if !strings.Contains(response, "TestServer") {
		t.Errorf("Expected response to contain 'TestServer', got: %s", response)
	}

	if !strings.Contains(response, "2.0") {
		t.Errorf("Expected response to contain JSON-RPC version '2.0', got: %s", response)
	}
}

func TestServer_ToolCall(t *testing.T) {
	srv := NewServer("TestServer", "1.0.0", "Desc")

	// Add a dummy tool to test the calling mechanism
	srv.AddTool(NewTool("ping", func(args map[string]interface{}) (*ToolResult, *RPCError) {
		return &ToolResult{
			MCPResult: MCPResult{ResultType: "complete"},
			Content:   []ToolContent{{Type: "text", Text: "pong"}},
		}, nil
	}))

	input := `{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": {"name": "ping", "arguments": {}}}` + "\n"
	stdin := bytes.NewBufferString(input)
	stdout := &bytes.Buffer{}

	go func() {
		srv.Serve(stdin, stdout)
	}()

	time.Sleep(100 * time.Millisecond)

	response := stdout.String()
	if !strings.Contains(response, "pong") {
		t.Errorf("Expected tool response 'pong', got: %s", response)
	}
}
