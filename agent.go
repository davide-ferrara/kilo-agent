package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"
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
	}
}

// Run is the agent actor loop: it consumes prompts from reqChan and streams
// the model output back as Message values on respChan. Requests are handled
// sequentially, so two Ask calls never overlap.
func (a *Agent) Run(reqChan <-chan string, respChan chan<- Message) {
	for prompt := range reqChan {
		a.Ask(ChatMessage{Role: RoleUser, Content: prompt}, respChan)
		respChan <- Message{MsgType: MsgGenerationDone}
	}
}

// Send request → receive the streamed messages → and iterate the internal round-trip loop
func (a *Agent) Ask(message ChatMessage, out chan<- Message) {
	a.Session.Messages = append(a.Session.Messages, message)
	a.loop(out)
}

func (a *Agent) loop(out chan<- Message) {
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

	reader := bufio.NewReader(resp.Body)
	assistantMessage := ChatMessage{Role: RoleAssistant}

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
		}

		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var bodyJSON ResponseChat
		err = json.Unmarshal(line, &bodyJSON)
		if err != nil {
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
			out <- Message{MsgThinking, bodyJSON.Message.Thinking}
		}
		if len(bodyJSON.Message.Content) > 0 {
			assistantMessage.Content += bodyJSON.Message.Content
			out <- Message{MsgResponse, bodyJSON.Message.Content}
		}
	}

	if assistantMessage.Content != "" || assistantMessage.Thinking != "" {
		a.Session.Messages = append(a.Session.Messages, assistantMessage)
	}
}

func (a *Agent) handleToolCall(toolCalls []ToolCall, out chan<- Message) {
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
		}[tool]
		if emoji == "" {
			emoji = "🔧"
		}
		out <- Message{MsgTool, emoji + " " + tool}

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
		default:
			log.Println("name: ", tool)
			toolResult = "No tools for that!"
		}

		a.Session.Messages = append(a.Session.Messages,
			ChatMessage{Role: RoleTool, Content: toolResult})
	}
}
