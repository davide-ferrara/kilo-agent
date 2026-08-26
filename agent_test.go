package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestAskStreamsThinkingAndStoresAssistantResponse(t *testing.T) {
	agent := NewAgent("test", "system")
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

	out := make(chan Message, 4)
	agent.Ask(ChatMessage{Role: RoleUser, Content: "Hi"}, out)
	close(out)

	var thinking strings.Builder
	var output strings.Builder
	for message := range out {
		if message.MsgType == MsgThinking {
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
