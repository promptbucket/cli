package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/promptbucket/cli/internal/persona"
)

// Request is an incoming JSON-RPC 2.0 message.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is an outgoing JSON-RPC 2.0 message.
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError is a JSON-RPC error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Server is the MCP server.
type Server struct {
	personaDir   string
	p            *persona.Persona
	systemPrompt string
}

// New creates a new MCP server for the given persona directory.
func New(dir string) (*Server, error) {
	p, err := persona.Load(dir)
	if err != nil {
		return nil, err
	}
	sp, err := persona.SystemPrompt(dir, p)
	if err != nil {
		return nil, err
	}
	return &Server{personaDir: dir, p: p, systemPrompt: sp}, nil
}

// Serve reads JSON-RPC requests from stdin and writes responses to stdout.
func (s *Server) Serve() error {
	fmt.Fprintf(os.Stderr, "PromptBucket MCP server started — persona: %s\n", s.p.Name)
	fmt.Fprintf(os.Stderr, "Waiting for MCP client connection...\n")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer

	encoder := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			s.writeError(encoder, nil, -32700, "parse error")
			continue
		}

		switch req.Method {
		case "initialize":
			s.handleInitialize(encoder, req)
		case "resources/list":
			s.handleResourcesList(encoder, req)
		case "resources/read":
			s.handleResourcesRead(encoder, req)
		case "tools/list":
			s.handleToolsList(encoder, req)
		case "tools/call":
			s.handleToolsCall(encoder, req)
		default:
			s.writeError(encoder, req.ID, -32601, fmt.Sprintf("method not found: %s", req.Method))
		}
	}

	return scanner.Err()
}

func (s *Server) handleInitialize(enc *json.Encoder, req Request) {
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"resources": map[string]interface{}{},
			"tools":     map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "promptbucket",
			"version": "0.3.0",
		},
	}
	s.writeResult(enc, req.ID, result)
}

func (s *Server) handleResourcesList(enc *json.Encoder, req Request) {
	resources := []map[string]interface{}{
		{
			"uri":         "persona://system-prompt",
			"name":        fmt.Sprintf("%s — system prompt", s.p.Name),
			"description": fmt.Sprintf("Effective system prompt for the %s persona. Includes identity, instructions, and seed memories.", s.p.Identity.Role),
			"mimeType":    "text/markdown",
		},
		{
			"uri":         "persona://identity",
			"name":        fmt.Sprintf("%s — identity", s.p.Name),
			"description": "Raw identity fields from persona.yaml as JSON.",
			"mimeType":    "application/json",
		},
	}
	s.writeResult(enc, req.ID, map[string]interface{}{"resources": resources})
}

func (s *Server) handleResourcesRead(enc *json.Encoder, req Request) {
	var params struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.writeError(enc, req.ID, -32602, "invalid params")
		return
	}

	switch params.URI {
	case "persona://system-prompt":
		s.writeResult(enc, req.ID, map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"uri":      params.URI,
					"mimeType": "text/markdown",
					"text":     s.systemPrompt,
				},
			},
		})
	case "persona://identity":
		identityJSON, _ := json.Marshal(s.p.Identity)
		s.writeResult(enc, req.ID, map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"uri":      params.URI,
					"mimeType": "application/json",
					"text":     string(identityJSON),
				},
			},
		})
	default:
		s.writeError(enc, req.ID, -32602, fmt.Sprintf("unknown resource URI: %s", params.URI))
	}
}

func (s *Server) handleToolsList(enc *json.Encoder, req Request) {
	tools := []map[string]interface{}{
		{
			"name":        "save_memory",
			"description": "Save a memory from this session. Call when you learn a project fact, user preference, decision, or reusable procedure worth keeping for future sessions.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{
						"type":        "string",
						"description": "The memory text. 1-3 sentences, self-contained.",
					},
					"layer": map[string]interface{}{
						"type": "string",
						"enum": []string{"episodic", "semantic", "procedural"},
					},
					"importance": map[string]interface{}{
						"type":        "number",
						"minimum":     0,
						"maximum":     1,
						"description": "0.0–1.0. Critical facts: 0.9. Useful context: 0.7. Minor notes: 0.5.",
					},
				},
				"required": []string{"content", "layer", "importance"},
			},
		},
	}
	s.writeResult(enc, req.ID, map[string]interface{}{"tools": tools})
}

// saveMemoryArgs is the input for the save_memory tool.
type saveMemoryArgs struct {
	Content    string   `json:"content"`
	Layer      string   `json:"layer"`
	Importance *float64 `json:"importance"`
}

var validMemoryLayers = map[string]bool{
	"episodic":   true,
	"semantic":   true,
	"procedural": true,
}

// makeMemorySlug generates a filename slug from the first up to 5 words of content.
// Non-alphanumeric characters are stripped; words are joined with hyphens.
func makeMemorySlug(content string) string {
	words := strings.Fields(content)
	if len(words) > 5 {
		words = words[:5]
	}
	var parts []string
	for _, w := range words {
		w = strings.ToLower(w)
		var clean []rune
		for _, r := range w {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				clean = append(clean, r)
			}
		}
		if len(clean) > 0 {
			parts = append(parts, string(clean))
		}
	}
	return strings.Join(parts, "-")
}

// handleSaveMemory writes a scored memory .md file to .promptbucket/memory/<layer>/.
func (s *Server) handleSaveMemory(rawArgs json.RawMessage) (interface{}, error) {
	var args saveMemoryArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	if strings.TrimSpace(args.Content) == "" {
		return nil, fmt.Errorf("content, layer, and importance are required")
	}
	if args.Importance == nil {
		return nil, fmt.Errorf("content, layer, and importance are required")
	}
	if !validMemoryLayers[args.Layer] {
		return nil, fmt.Errorf("layer must be episodic, semantic, or procedural")
	}
	if *args.Importance < 0.0 || *args.Importance > 1.0 {
		return nil, fmt.Errorf("importance must be between 0.0 and 1.0")
	}

	now := time.Now().UTC()
	slug := makeMemorySlug(args.Content)
	var filename string
	if slug == "" {
		filename = now.Format("2006-01-02-150405") + ".md"
	} else {
		filename = fmt.Sprintf("%s-%s.md", now.Format("2006-01-02-150405"), slug)
	}

	memDir := filepath.Join(s.personaDir, "memory", args.Layer)
	if err := os.MkdirAll(memDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to write memory: %w", err)
	}

	body := fmt.Sprintf(
		"---\nimportance: %.1f\nlayer: %s\ncreated: %s\nlast_accessed: %s\n---\n\n%s\n",
		*args.Importance,
		args.Layer,
		now.Format(time.RFC3339),
		now.Format(time.RFC3339),
		strings.TrimSpace(args.Content),
	)

	filePath := filepath.Join(memDir, filename)
	if err := os.WriteFile(filePath, []byte(body), 0644); err != nil {
		return nil, fmt.Errorf("failed to write memory: %w", err)
	}

	return map[string]interface{}{
		"saved": true,
		"path":  filepath.Join("memory", args.Layer, filename),
	}, nil
}

// handleToolsCall dispatches tool invocations by name.
func (s *Server) handleToolsCall(enc *json.Encoder, req Request) {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.writeError(enc, req.ID, -32602, "invalid params")
		return
	}

	switch params.Name {
	case "save_memory":
		result, err := s.handleSaveMemory(params.Arguments)
		if err != nil {
			s.writeError(enc, req.ID, -32603, err.Error())
			return
		}
		res, ok := result.(map[string]interface{})
		if !ok {
			s.writeError(enc, req.ID, -32603, "internal error: unexpected result type")
			return
		}
		s.writeResult(enc, req.ID, map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": fmt.Sprintf("Memory saved: %v", res["path"])},
			},
		})
	default:
		s.writeError(enc, req.ID, -32601, fmt.Sprintf("unknown tool: %s", params.Name))
	}
}

func (s *Server) writeResult(enc *json.Encoder, id interface{}, result interface{}) {
	enc.Encode(Response{JSONRPC: "2.0", ID: id, Result: result})
}

func (s *Server) writeError(enc *json.Encoder, id interface{}, code int, msg string) {
	enc.Encode(Response{JSONRPC: "2.0", ID: id, Error: &RPCError{Code: code, Message: msg}})
}
