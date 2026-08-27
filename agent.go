package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"kilo-agent/tui"
)

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
	RoleTool      = "tool"
)

type ChatMessage struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	Thinking  string     `json:"thinking,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	Arguments   json.RawMessage `json:"arguments,omitempty"`
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolCall struct {
	Function ToolFunction `json:"function"`
}

type ToolCallChat struct {
	Role      string     `json:"role"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls"`
}

type Chat struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Tools    []Tool        `json:"tools"`
	Stream   bool          `json:"stream"`
}

type ResponseChat struct {
	Model              string      `json:"model"`
	CreatedAt          string      `json:"created_at"`
	Message            ChatMessage `json:"message"`
	DoneReason         string      `json:"done_reason"`
	Done               bool        `json:"done"`
	TotalDuration      int64       `json:"total_duration"`
	LoadDuration       int64       `json:"load_duration"`
	PromptEvalCount    int         `json:"prompt_eval_count"`
	PromptEvalDuration int64       `json:"prompt_eval_duration"`
	EvalCount          int         `json:"eval_count"`
	EvalDuration       int64       `json:"eval_duration"`
}

type Agent struct {
	Name         string
	SystemPrompt string
	Session      Chat // TODO: Add Session obj
	Client       http.Client
	ContextSize  int
}

func NewAgent(name string, systemPrompt string) *Agent {
	return &Agent{
		Name: name,
		Session: Chat{
			Model:    "qwen3:14b",
			Messages: []ChatMessage{{Role: RoleSystem, Content: systemPrompt}},
			Tools:    RegisterTools(),
			Stream:   true,
		},
		Client: http.Client{
			Timeout: time.Minute * 2,
		},
		ContextSize: 32768,
	}
}

// Run is the agent actor loop: it consumes prompts from reqChan and streams
// the model output back as Message values on respChan. Requests are handled
// sequentially, so two Ask calls never overlap.
func (a *Agent) Run(reqChan <-chan string, respChan chan<- tui.Message) {
	for prompt := range reqChan {
		func() {
			defer func() {
				if value := recover(); value != nil {
					log.Printf("agent request failed: %v", value)
					respChan <- tui.Message{MsgType: tui.MsgError, Data: "The model request failed. Check /tmp/kilo-agent.log and make sure Ollama is running."}
				}
			}()
			a.Ask(ChatMessage{Role: RoleUser, Content: prompt}, respChan)
		}()
		respChan <- tui.Message{MsgType: tui.MsgGenerationDone}
	}
}

// Send request → receive the streamed messages → and iterate the internal round-trip loop
func (a *Agent) Ask(message ChatMessage, out chan<- tui.Message) {
	a.Session.Messages = append(a.Session.Messages, message)
	a.loop(out)
}

func (a *Agent) loop(out chan<- tui.Message) {
	jsonData, err := json.Marshal(a.Session)
	if err != nil {
		panic(err)
	}

	log.Println(a.Session.Messages)

	req, err := http.NewRequest(
		"POST",
		"http://localhost:11434/api/chat",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		log.Println(err)
		panic(err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := a.Client.Do(req)
	if err != nil {
		log.Println(err)
		panic(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4*1024))
		panic(fmt.Errorf("ollama returned %s: %s", resp.Status, bytes.TrimSpace(body)))
	}

	reader := bufio.NewReader(resp.Body)
	assistantMessage := ChatMessage{Role: RoleAssistant}
	usage := tui.Usage{ContextSize: a.ContextSize}

	for {
		line, readErr := reader.ReadBytes('\n')

		if len(bytes.TrimSpace(line)) > 0 {
			var bodyJSON ResponseChat
			if err := json.Unmarshal(line, &bodyJSON); err != nil {
				log.Println(err)
				panic(err)
			}

			if len(bodyJSON.Message.ToolCalls) > 0 {
				a.handleToolCall(bodyJSON.Message.ToolCalls, out)
				a.loop(out)
				return
			}

			if len(bodyJSON.Message.Thinking) > 0 {
				assistantMessage.Thinking += bodyJSON.Message.Thinking
				out <- tui.Message{MsgType: tui.MsgThinking, Data: bodyJSON.Message.Thinking}
			}
			if len(bodyJSON.Message.Content) > 0 {
				assistantMessage.Content += bodyJSON.Message.Content
				out <- tui.Message{MsgType: tui.MsgResponse, Data: bodyJSON.Message.Content}
			}
			if bodyJSON.PromptEvalCount > 0 {
				usage.PromptTokens = bodyJSON.PromptEvalCount
			}
			if bodyJSON.EvalCount > 0 {
				usage.OutputTokens = bodyJSON.EvalCount
				if bodyJSON.EvalDuration > 0 {
					usage.TokensPerSec = float64(bodyJSON.EvalCount) / (float64(bodyJSON.EvalDuration) / float64(time.Second))
				}
			}
		}

		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			panic(readErr)
		}
	}

	if assistantMessage.Content != "" || assistantMessage.Thinking != "" {
		a.Session.Messages = append(a.Session.Messages, assistantMessage)
	}
	if usage.PromptTokens > 0 || usage.OutputTokens > 0 {
		out <- tui.Message{MsgType: tui.MsgUsage, Data: usage}
	}
}

func (a *Agent) handleToolCall(toolCalls []ToolCall, out chan<- tui.Message) {
	a.Session.Messages = append(a.Session.Messages,
		ChatMessage{Role: RoleAssistant, ToolCalls: toolCalls})

	for i := range toolCalls {
		tool := toolCalls[i].Function.Name

		emoji := map[string]string{
			"random_int":   "🎲",
			"random_int_n": "🎲",
			"read_file":    "🔨",
			"write_file":   "✍️",
			"pwd":          "📍",
			"ls":           "📁",
			"exec_cmd":     "⚡",
			"web_search":   "🌐",
		}[tool]
		if emoji == "" {
			emoji = "🔧"
		}
		out <- tui.Message{MsgType: tui.MsgTool, Data: emoji + " " + tool}

		var toolResult string
		switch tool {
		case "random_int":
			toolResult = RandomInt()
		case "random_int_n":
			var args struct {
				N int `json:"n"`
			}
			if len(toolCalls[i].Function.Arguments) > 0 {
				if err := json.Unmarshal(toolCalls[i].Function.Arguments, &args); err != nil {
					log.Println("parse args: ", err)
					toolResult = "Error: invalid arguments"
					break
				}
			}
			toolResult = RandomIntN(args.N)
		case "read_file":
			var args struct {
				Name string `json:"name"`
			}
			if len(toolCalls[i].Function.Arguments) > 0 {
				if err := json.Unmarshal(toolCalls[i].Function.Arguments, &args); err != nil {
					log.Println("parse args: ", err)
					toolResult = "Error: invalid arguments"
					break
				}
			}
			toolResult = ReadFile(args.Name)
		case "write_file":
			var args struct {
				Path    string `json:"path"`
				Content string `json:"content"`
			}
			if len(toolCalls[i].Function.Arguments) > 0 {
				if err := json.Unmarshal(toolCalls[i].Function.Arguments, &args); err != nil {
					log.Println("parse args: ", err)
					toolResult = "Error: invalid arguments"
					break
				}
			}
			toolResult = WriteFile(args.Path, args.Content)
		case "pwd":
			toolResult = Pwd()
		case "ls":
			toolResult = Ls()
		case "exec_cmd":
			var args struct {
				Name string   `json:"name"`
				Args []string `json:"args"`
			}
			if len(toolCalls[i].Function.Arguments) > 0 {
				if err := json.Unmarshal(toolCalls[i].Function.Arguments, &args); err != nil {
					log.Println("parse args: ", err)
					toolResult = "Error: invalid arguments"
					break
				}
			}
			toolResult = ExecCmd(args.Name, args.Args...)
		case "web_search":
			var args struct {
				Query string `json:"query"`
				Limit int    `json:"limit"`
				Type  string `json:"type"`
			}
			if len(toolCalls[i].Function.Arguments) > 0 {
				if err := json.Unmarshal(toolCalls[i].Function.Arguments, &args); err != nil {
					log.Println("parse args: ", err)
					toolResult = "Error: invalid arguments"
					break
				}
			}
			result, err := WebSearchJSON(nil, args.Query, args.Limit, args.Type)
			if err != nil {
				toolResult = "Error: " + err.Error()
				break
			}
			toolResult = result
		default:
			log.Println("name: ", tool)
			toolResult = "No tools for that!"
		}

		a.Session.Messages = append(a.Session.Messages,
			ChatMessage{Role: RoleTool, Content: toolResult})
	}
}
