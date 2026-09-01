package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestExec(t *testing.T) {
	cmd := exec.Command("/usr/bin/echo", "OK")
	if errors.Is(cmd.Err, exec.ErrDot) {
		cmd.Err = nil
	}
	out, err := cmd.Output()
	if err != nil {
		t.Log("could not run command: ", err)
	}

	t.Log(string(out))
	if string(out) != "OK\n" {
		t.Fatal("Command output do not match.")
	}
}

func TestExecGrep(t *testing.T) {
	cmd := exec.Command("/usr/bin/grep", "-r", "agent", ".")
	if errors.Is(cmd.Err, exec.ErrDot) {
		cmd.Err = nil
	}
	_, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
}

func TestEdit(t *testing.T) {
	file, err := os.ReadFile("./test/edit_test.txt")
	if err != nil {
		t.Fatal(err)
	}
	oldString := []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do\neiusmod tempor incididunt ut labore et dolore magna aliqua.")
	if bytes.Count(file, oldString) < 1 {
		t.Fatal("OldString not found")
	}
	// bytes.Replace(file, oldString, , n int)
}

func TestWebFetch(t *testing.T) {
	if os.Getenv("KILO_INTEGRATION_TESTS") == "" {
		t.Skip("set KILO_INTEGRATION_TESTS=1 to run live web checks")
	}
	endpoint := "https://api.duckduckgo.com/?q=" + url.QueryEscape("Alan Turing") + "&format=json"
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept", "application/json")

	client := http.Client{
		Timeout: 10 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Status code: ", res.StatusCode)

	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal("impossible to read all body of response: ", err)
	}
	fmt.Printf("res body: %s", string(resBody))
}

func TestBingRss(t *testing.T) {
	if os.Getenv("KILO_INTEGRATION_TESTS") == "" {
		t.Skip("set KILO_INTEGRATION_TESTS=1 to run live web checks")
	}
	client := &http.Client{Timeout: 10 * time.Second}
	out, err := WebSearchJSON(client, "Milan news", 2, "news")
	if err != nil {
		t.Fatal(err)
	}
	var pages []WebPage
	if err := json.Unmarshal([]byte(out), &pages); err != nil {
		t.Fatal(err)
	}
	if len(pages) == 0 {
		t.Fatal("no pages")
	}
	log.Printf("got %d pages; first: %q\n%s", len(pages), pages[0].Link, out)
}

func TestWebSearchToolIsRegistered(t *testing.T) {
	for _, tool := range RegisterTools() {
		if tool.Function.Name != "web_search" {
			continue
		}
		var schema struct {
			Required []string `json:"required"`
		}
		if err := json.Unmarshal(tool.Function.Parameters, &schema); err != nil {
			t.Fatal(err)
		}
		if len(schema.Required) != 1 || schema.Required[0] != "query" {
			t.Fatalf("web_search should require query, got %v", schema.Required)
		}
		return
	}
	t.Fatal("web_search tool is not registered")
}

func TestTelegramIsBotConfiguredToolIsRegistered(t *testing.T) {
	want := map[string]bool{
		"telegram_is_bot_configured": false,
		"telegram_start_pairing":     false,
		"telegram_complete_pairing":  false,
		"telegram_send_message":      false,
	}

	for _, tool := range RegisterTools() {
		if _, ok := want[tool.Function.Name]; ok {
			want[tool.Function.Name] = true
		}
	}
	for name, registered := range want {
		if !registered {
			t.Errorf("%s tool is not registered", name)
		}
	}
}

func TestTelegramPairingAndSendMessage(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)

	agent := NewAgent(Config{
		Name:             "test",
		TelegramBotToken: "test-token",
	}, "system")

	const chatID int64 = 765046979
	messageSent := false
	agent.TelegramClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case strings.HasSuffix(req.URL.Path, "/getMe"):
			return telegramToolHTTPResponse(`{
				"ok": true,
				"result": {
					"id": 42,
					"is_bot": true,
					"first_name": "Test Bot",
					"username": "test_bot"
				}
			}`), nil
		case strings.HasSuffix(req.URL.Path, "/getUpdates"):
			if agent.telegramStart == nil {
				t.Fatal("getUpdates called without pending pairing state")
			}
			body := fmt.Sprintf(`{
				"ok": true,
				"result": [{
					"update_id": 100,
					"message": {
						"message_id": 7,
						"date": 123456,
						"chat": {"id": %d, "type": "private"},
						"text": "/start %d"
					}
				}]
			}`, chatID, agent.telegramStart.Otp)
			return telegramToolHTTPResponse(body), nil
		case strings.HasSuffix(req.URL.Path, "/sendMessage"):
			var payload struct {
				ChatID int64  `json:"chat_id"`
				Text   string `json:"text"`
			}
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Fatalf("decode sendMessage payload: %v", err)
			}
			if payload.ChatID != chatID {
				t.Errorf("sent chat ID = %d, want %d", payload.ChatID, chatID)
			}
			if payload.Text != "Ciao!" {
				t.Errorf("sent text = %q, want %q", payload.Text, "Ciao!")
			}
			messageSent = true
			return telegramToolHTTPResponse(fmt.Sprintf(`{
				"ok": true,
				"result": {
					"message_id": 8,
					"date": 123456,
					"chat": {"id": %d, "type": "private"},
					"text": "Ciao!"
				}
			}`, chatID)), nil
		default:
			t.Fatalf("unexpected Telegram request: %s", req.URL)
			return nil, nil
		}
	})

	startResult := agent.TelegramStartPairing()
	if !strings.Contains(startResult, "https://t.me/test_bot?start=") {
		t.Fatalf("TelegramStartPairing() = %q, want pairing link", startResult)
	}

	if got := agent.TelegramCompletePairing(); got != "Telegram chat paired successfully." {
		t.Fatalf("TelegramCompletePairing() = %q", got)
	}
	if agent.Config.TelegramChatID != chatID {
		t.Errorf("agent chat ID = %d, want %d", agent.Config.TelegramChatID, chatID)
	}

	saved, err := loadConfig()
	if err != nil {
		t.Fatalf("load saved config: %v", err)
	}
	if saved.TelegramChatID != chatID {
		t.Errorf("saved chat ID = %d, want %d", saved.TelegramChatID, chatID)
	}

	if got := agent.TelegramSendMessage("Ciao!"); got != "Telegram message sent successfully." {
		t.Fatalf("TelegramSendMessage() = %q", got)
	}
	if !messageSent {
		t.Error("sendMessage request was not made")
	}
}

func TestTelegramStartPairingReusesPendingLink(t *testing.T) {
	agent := NewAgent(Config{
		Name:             "test",
		TelegramBotToken: "test-token",
	}, "system")

	getMeCalls := 0
	agent.TelegramClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if !strings.HasSuffix(req.URL.Path, "/getMe") {
			t.Fatalf("unexpected Telegram request: %s", req.URL)
		}
		getMeCalls++
		return telegramToolHTTPResponse(`{
			"ok": true,
			"result": {
				"id": 42,
				"is_bot": true,
				"first_name": "Test Bot",
				"username": "test_bot"
			}
		}`), nil
	})

	first := agent.TelegramStartPairing()
	if agent.telegramStart == nil {
		t.Fatal("TelegramStartPairing() did not retain pairing state")
	}
	wantURL := agent.telegramStart.Url
	second := agent.TelegramStartPairing()

	if !strings.Contains(first, wantURL) || !strings.Contains(second, wantURL) {
		t.Fatalf("pairing results do not contain the same link: first=%q second=%q", first, second)
	}
	if getMeCalls != 1 {
		t.Errorf("getMe calls = %d, want 1", getMeCalls)
	}
}

func telegramToolHTTPResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestTelegramIsBotConfigured(t *testing.T) {
	tests := []struct {
		name   string
		token  string
		chatID int64
		want   string
	}{
		{
			name:   "configured and paired",
			token:  "test-token",
			chatID: 765046979,
			want:   "Telegram bot and chat are configured and ready to use.",
		},
		{
			name:  "configured but not paired",
			token: "test-token",
			want:  "Telegram bot token is configured, but no chat is paired. Call telegram_start_pairing.",
		},
		{
			name: "not configured",
			want: "Telegram bot is not configured. Set telegram_bot_token in config.json.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := NewAgent(Config{
				Name:             "test",
				TelegramBotToken: tt.token,
				TelegramChatID:   tt.chatID,
			}, "system")

			if got := agent.TelegramIsBotConfigured(); got != tt.want {
				t.Errorf("TelegramIsBotConfigured() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWebSearchUsesNewsEndpoint(t *testing.T) {
	var requestedPath string
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestedPath = req.URL.Path
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body: io.NopCloser(strings.NewReader(`
				<rss><channel><item>
				<title>Local headline</title>
				<link>https://example.com/story</link>
				<description>Local news</description>
				<pubDate>Thu, 27 Aug 2026 12:00:00 GMT</pubDate>
				</item></channel></rss>`)),
			Header: make(http.Header),
		}, nil
	})}

	pages, err := WebSearch(client, "local headline", 5, "news")
	if err != nil {
		t.Fatal(err)
	}
	if requestedPath != "/news/search" {
		t.Fatalf("expected news endpoint, got %q", requestedPath)
	}
	if len(pages) != 1 || pages[0].Title != "Local headline" {
		t.Fatalf("unexpected results: %#v", pages)
	}
}
