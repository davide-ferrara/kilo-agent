package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"kilo-agent/tui"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestAskStreamsThinkingAndStoresAssistantResponse(t *testing.T) {
	agent := NewAgent(Config{Name: "test"}, "system")
	agent.Client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		stream := strings.Join([]string{
			`{"message":{"role":"assistant","thinking":"private reasoning"}}`,
			`{"message":{"role":"assistant","content":"Hello"}}`,
			`{"message":{"role":"assistant","content":" David"},"done":true}`,
		}, "\n") + "\n"
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(stream)),
			Header:     make(http.Header),
		}, nil
	})

	out := make(chan tui.Message, 4)
	agent.Ask(ChatMessage{Role: RoleUser, Content: "Hi"}, out)
	close(out)

	var thinking strings.Builder
	var output strings.Builder
	for message := range out {
		if message.MsgType == tui.MsgThinking {
			thinking.WriteString(message.Data.(string))
			continue
		}
		output.WriteString(message.Data.(string))
	}
	if thinking.String() != "private reasoning" {
		t.Fatalf("Ask() thinking = %q, want %q", thinking.String(), "private reasoning")
	}
	if output.String() != "Hello David" {
		t.Fatalf("Ask() output = %q, want %q", output.String(), "Hello David")
	}

	last := agent.Session.Messages[len(agent.Session.Messages)-1]
	if last.Role != RoleAssistant || last.Content != "Hello David" {
		t.Fatalf("stored assistant message = %#v", last)
	}
}

func TestAskProcessesFinalStreamRecordWithoutNewline(t *testing.T) {
	agent := NewAgent(Config{Name: "test"}, "system")
	agent.Client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"message":{"role":"assistant","content":"complete"},"done":true}`)),
			Header:     make(http.Header),
		}, nil
	})

	out := make(chan tui.Message, 1)
	agent.Ask(ChatMessage{Role: RoleUser, Content: "Hi"}, out)
	message := <-out
	if message.MsgType != tui.MsgResponse || message.Data != "complete" {
		t.Fatalf("final stream record = %#v", message)
	}
}

func TestNewAgentStoresConfig(t *testing.T) {
	config := Config{
		Name:             "Configured Agent",
		TelegramBotToken: "test-token",
	}

	agent := NewAgent(config, "system prompt")

	if agent.Config != config {
		t.Errorf("agent config = %#v, want %#v", agent.Config, config)
	}
	if agent.Name != config.Name {
		t.Errorf("agent name = %q, want %q", agent.Name, config.Name)
	}
	if agent.SystemPrompt != "system prompt" {
		t.Errorf("system prompt = %q, want %q", agent.SystemPrompt, "system prompt")
	}
}

func TestTelegramBot(t *testing.T) {
	agent := NewAgent(Config{
		Name:             "test",
		TelegramBotToken: "test-token",
	}, "system")

	bot, err := agent.telegramBot()
	if err != nil {
		t.Fatalf("telegramBot() error = %v", err)
	}
	if bot.Token != "test-token" {
		t.Errorf("bot token = %q, want %q", bot.Token, "test-token")
	}
	if bot.Client == nil {
		t.Fatal("bot client is nil")
	}
	if bot.Client.Timeout != 30*time.Second {
		t.Errorf("bot timeout = %v, want %v", bot.Client.Timeout, 30*time.Second)
	}
}

func TestTelegramBotNotConfigured(t *testing.T) {
	agent := NewAgent(Config{Name: "test"}, "system")

	bot, err := agent.telegramBot()
	if err == nil {
		t.Fatal("telegramBot() error = nil, want an error")
	}
	if bot != nil {
		t.Errorf("telegramBot() = %#v, want nil", bot)
	}
}
