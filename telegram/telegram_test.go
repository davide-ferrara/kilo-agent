package telegram

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestBot(t *testing.T, handler roundTripFunc) Bot {
	t.Helper()
	return Bot{
		Client: &http.Client{Transport: handler},
		Token:  "test-token",
	}
}

func jsonHTTPResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestGetMe(t *testing.T) {
	bot := newTestBot(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("method = %q, want %q", req.Method, http.MethodGet)
		}
		if req.URL.String() != TelegramAPI+"test-token/getMe" {
			t.Fatalf("URL = %q, want %q", req.URL.String(), TelegramAPI+"test-token/getMe")
		}

		return jsonHTTPResponse(`{
			"ok": true,
			"result": {
				"id": 42,
				"is_bot": true,
				"first_name": "Test Bot",
				"username": "test_bot"
			}
		}`), nil
	})

	user, err := bot.GetMe()
	if err != nil {
		t.Fatalf("GetMe() error = %v", err)
	}
	if user.ID != 42 {
		t.Errorf("user ID = %d, want 42", user.ID)
	}
	if user.Username == nil || *user.Username != "test_bot" {
		t.Errorf("username = %v, want test_bot", user.Username)
	}
}

func TestGetMeReturnsTelegramError(t *testing.T) {
	bot := newTestBot(t, func(*http.Request) (*http.Response, error) {
		return jsonHTTPResponse(`{"ok":false,"description":"Unauthorized"}`), nil
	})

	_, err := bot.GetMe()
	if err == nil || !strings.Contains(err.Error(), "Unauthorized") {
		t.Fatalf("GetMe() error = %v, want an Unauthorized error", err)
	}
}

func TestStart(t *testing.T) {
	bot := newTestBot(t, func(*http.Request) (*http.Response, error) {
		return jsonHTTPResponse(`{
			"ok": true,
			"result": {
				"id": 42,
				"is_bot": true,
				"first_name": "Test Bot",
				"username": "test_bot"
			}
		}`), nil
	})

	start, err := bot.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if start.Otp < 0 || start.Otp >= 8192 {
		t.Errorf("OTP = %d, want a value in [0, 8192)", start.Otp)
	}
	wantURL := "https://t.me/test_bot?start=" + strconv.Itoa(start.Otp)
	if start.Url != wantURL {
		t.Errorf("URL = %q, want %q", start.Url, wantURL)
	}
}

func TestGetUpdates(t *testing.T) {
	bot := newTestBot(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("method = %q, want %q", req.Method, http.MethodGet)
		}
		if req.URL.String() != TelegramAPI+"test-token/getUpdates" {
			t.Fatalf("URL = %q, want %q", req.URL.String(), TelegramAPI+"test-token/getUpdates")
		}

		return jsonHTTPResponse(`{
			"ok": true,
			"result": [{
				"update_id": 100,
				"message": {
					"message_id": 7,
					"date": 123456,
					"chat": {"id": 765046979, "type": "private"},
					"text": "/start 1234"
				}
			}]
		}`), nil
	})

	updates, err := bot.GetUpdates()
	if err != nil {
		t.Fatalf("GetUpdates() error = %v", err)
	}
	if len(updates) != 1 {
		t.Fatalf("len(updates) = %d, want 1", len(updates))
	}
	if updates[0].UpdateId != 100 {
		t.Errorf("update ID = %d, want 100", updates[0].UpdateId)
	}
	if updates[0].Message.Chat.ID != 765046979 {
		t.Errorf("chat ID = %d, want 765046979", updates[0].Message.Chat.ID)
	}
}

func TestGetChatID(t *testing.T) {
	matchingText := "/start 1234"
	unrelatedText := "the code 1234 appears here"
	updates := []Update{
		{Message: Message{Chat: Chat{ID: 1}, Text: nil}},
		{Message: Message{Chat: Chat{ID: 2}, Text: &unrelatedText}},
		{Message: Message{Chat: Chat{ID: 765046979}, Text: &matchingText}},
	}

	chatID, err := GetChatID(Start{Otp: 1234}, updates)
	if err != nil {
		t.Fatalf("GetChatID() error = %v", err)
	}
	if chatID != 765046979 {
		t.Errorf("GetChatID() = %d, want 765046979", chatID)
	}
}

func TestGetChatIDNotFound(t *testing.T) {
	text := "/start 9999"
	updates := []Update{{Message: Message{Chat: Chat{ID: 42}, Text: &text}}}

	chatID, err := GetChatID(Start{Otp: 1234}, updates)
	if err == nil {
		t.Fatal("GetChatID() error = nil, want an error")
	}
	if chatID != 0 {
		t.Errorf("GetChatID() = %d, want 0 on error", chatID)
	}
}

func TestSendMessage(t *testing.T) {
	bot := newTestBot(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("method = %q, want %q", req.Method, http.MethodPost)
		}
		if req.URL.String() != TelegramAPI+"test-token/sendMessage" {
			t.Fatalf("URL = %q, want %q", req.URL.String(), TelegramAPI+"test-token/sendMessage")
		}
		if got := req.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", got)
		}

		var payload SendMessagePayload
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if payload.ChatID != 765046979 {
			t.Errorf("payload chat ID = %d, want 765046979", payload.ChatID)
		}
		if payload.Text != "Ciao!" {
			t.Errorf("payload text = %q, want Ciao!", payload.Text)
		}

		return jsonHTTPResponse(`{
			"ok": true,
			"result": {
				"message_id": 8,
				"date": 123456,
				"chat": {"id": 765046979, "type": "private"},
				"text": "Ciao!"
			}
		}`), nil
	})

	message, err := bot.SendMessage(765046979, "Ciao!")
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	if message.MessageId != 8 {
		t.Errorf("message ID = %d, want 8", message.MessageId)
	}
	if message.Text == nil || *message.Text != "Ciao!" {
		t.Errorf("message text = %v, want Ciao!", message.Text)
	}
}

func TestHTTPClientError(t *testing.T) {
	wantErr := errors.New("transport failed")
	bot := newTestBot(t, func(*http.Request) (*http.Response, error) {
		return nil, wantErr
	})

	_, err := bot.GetUpdates()
	if err == nil {
		t.Fatal("GetUpdates() error = nil, want an error")
	}
	if got := err.Error(); got != "Client Error" {
		t.Errorf("GetUpdates() error = %q, want %q", got, "Client Error")
	}
}
