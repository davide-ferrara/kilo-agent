package main

import (
	"encoding/json"
	"errors"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func RegisterTools() []Tool {
	tools := []Tool{}
	randomIntTool := Tool{
		Type: "function",
		Function: ToolFunction{
			Name: "random_int",
			Description: "Returns a random integer",
		},
	}

	randomIntNTool := Tool{
		Type: "function",
		Function: ToolFunction{
			Name: "random_int_n",
			Description: "Returns, as an int, a non-negative pseudo-random number in the half-open interval [0,n). It panics if n <= 0.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"n": {"type": "integer", "description": "The upper bound (exclusive) for the random number"}
				},
				"required": ["n"]
			}`),
		},
	}
	
	readFileTool := Tool{
		Type: "function",
		Function: ToolFunction{
			Name: "read_file",
			Description: "Read a file and returns it's content as a string.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"name": {"type": "string", "description": "The name of the file"}
				},
				"required": ["name"]
			}`),
		},
	}

	tools = append(tools, randomIntTool, randomIntNTool, readFileTool)

	pwdTool := Tool{
		Type: "function",
		Function: ToolFunction{
			Name: "pwd",
			Description: "Returns the current working directory",
		},
	}

	tools = append(tools, pwdTool)

	lsTool := Tool{
		Type: "function",
		Function: ToolFunction{
			Name: "ls",
			Description: "List the entries (files and directories) in the current working directory",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {}
			}`),
		},
	}

	tools = append(tools, lsTool)

	writeFileTool := Tool{
		Type: "function",
		Function: ToolFunction{
			Name: "write_file",
			Description: "Write content to a file at the given path",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"path": {"type": "string", "description": "The path of the file to write"},
					"content": {"type": "string", "description": "The content to write"}
				},
				"required": ["path", "content"]
			}`),
		},
	}

	tools = append(tools, writeFileTool)

	execCmdTool := Tool{
		Type: "function",
		Function: ToolFunction{
			Name: "exec_cmd",
			Description: "Execute a system command and return its output",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"name": {"type": "string", "description": "The command to execute"},
					"args": {"type": "array", "items": {"type": "string"}, "description": "The arguments for the command"}
				},
				"required": ["name", "args"]
			}`),
		},
	}

	tools = append(tools, execCmdTool)

	return tools
}

func RandomInt() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return strconv.Itoa(r.Int())
}

func RandomIntN(n int) string {
	if n <= 0 { return "panic" }
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return strconv.Itoa(r.Intn(n))
}

func Pwd() string {
	dir, err := os.Getwd()
	if err != nil {
		return "error: " + err.Error()
	}
	return dir
}

func Ls() string {
	dir, err := os.Getwd()
	if err != nil {
		return "error: " + err.Error()
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "error: " + err.Error()
	}
	var sb strings.Builder
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		sb.WriteString(name)
		sb.WriteString("\n")
	}
	return sb.String()
}

func ReadFile(name string) string {
	path := Pwd() + "/" + name
	data, err := os.ReadFile(path)
	if err != nil {
		return "File doesn't exist or could not be read: " + err.Error()
	}
	return string(data)
}

func WriteFile(path string, content string) string {
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return err.Error()
	}
	return "success"
}

// FIX: This is not secure.
func ExecCmd(name string, args ...string) string {
	allArgs := append([]string{"30", name}, args...)
	cmd := exec.Command("timeout", allArgs...)
	if errors.Is(cmd.Err, exec.ErrDot) {
		cmd.Err = nil
	}
	out, err := cmd.Output()
	if err != nil {
		return "tool exec_cmd error: " + err.Error()
	}
	return string(out)
}
