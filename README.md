# Miniature Context Protocol

A lightweight, type-safe implementation of the Model Context Protocol (MCP) in Go.

## Features

- **Tool Registration**: Define tool logic using idiomatic Go functions.
- **Standard I/O**: Exposes a more lenient interpretation of the MCP Spec that still maintains compatibility with modern MCP clients.

## Quick Start

### 1. Define a Tool
Tools are created using `NewTool` and configured with `WithParams`.

```go
func FooTool(args map[string]interface{}) (*mcp.ToolResult, *mcp.RPCError) {
    name, ok := args["name"].(string)
    if !ok {
        return nil, &mcp.RPCError{Code: -32602, Message: "Invalid name parameter"}
    }

    return &mcp.ToolResult{
        MCPResult: mcp.MCPResult{ResultType: "complete"},
        Content: []mcp.ToolContent{
            {Type: "text", Text: fmt.Sprintf("Hello, %s!", name)},
        },
    }, nil
}

// Create the tool definition
fooTool := mcp.NewTool(
    "greet", 
    FooTool, 
    mcp.WithDescription("Greets a user by name"),
    mcp.WithParams(
        mcp.Param{Name: "name", Type: "string", Description: "User's name", Required: true},
    ),
)
```

### 2. Initialize and Run the Server
```go
func main() {
    srv := mcp.NewServer("FriendlyFoo", "1.0.0", "A helpful greeting server")
    srv.AddTool(greetTool)

    srv.Serve(os.Stdin, os.Stdout)
}
```

### 3. Manual Testing
You can test the server by piping JSON-RPC requests into your compiled binary:

```bash
echo '{"jsonrpc": "2.0", "id": 1, "method": "server/discover"}' | ./my-mcp-server
```

## Protocol Details

The server implements only the following methods:
- `server/discover`: Returns server metadata, capabilities, and supported versions.
- `tools/list`: Lists all registered tools and their JSON schemas.
- `tools/call`: Executes a specific tool by name with the provided arguments.
