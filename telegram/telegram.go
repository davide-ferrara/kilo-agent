package telegram

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
)

const (
	TelegramAPI = "https://api.telegram.org/bot"
)

// Returns the URL for the user to START the bot.
// NOTE: The user has to configure it, can't be automated.
func (b *Bot) Start() (Start, error) {
	res, err := b.GetMe()
	if err != nil {
		return Start{}, fmt.Errorf("Error GetMe(): %v\n", err)
	}

	// FIX: NOT SAFE
	otp := rand.IntN(8192)
	url := "https://t.me/" + *res.Username + "?start=" + strconv.Itoa(otp)

	return Start{Url: url, Otp: otp}, nil
}

func (b Bot) GetMe() (User, error) {
	endpoint := TelegramAPI + b.Token + "/getMe"

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return User{}, fmt.Errorf("GET Error")
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := b.Client.Do(req)
	if err != nil {
		return User{}, fmt.Errorf("Client Error")
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return User{}, fmt.Errorf("Body Error")
	}

	var response Response[User]

	if err := json.Unmarshal(body, &response); err != nil {
		return User{}, fmt.Errorf("JSON decode error: %w", err)
	}

	if !response.OK {
		return User{}, fmt.Errorf("Telegram error: %s", response.Description)
	}

	return response.Result, nil
}

func (b *Bot) GetUpdates() ([]Update, error) {
	endpoint := TelegramAPI + b.Token + "/getUpdates"

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return []Update{}, fmt.Errorf("GET Error")
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := b.Client.Do(req)
	if err != nil {
		return []Update{}, fmt.Errorf("Client Error")
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return []Update{}, fmt.Errorf("Body Error")
	}

	var response Response[[]Update]

	if err := json.Unmarshal(body, &response); err != nil {
		return []Update{}, fmt.Errorf("JSON decode error: %w", err)
	}

	if !response.OK {
		return []Update{}, fmt.Errorf("Telegram error: %s", response.Description)
	}

	return response.Result, nil
}

func GetChatID(start Start, updates []Update) (int64, error) {
	expected := "/start " + strconv.Itoa(start.Otp)

	for i := range updates {
		message := updates[i].Message

		if message.Text == nil {
			continue
		}

		if *message.Text == expected {
			return message.Chat.ID, nil
		}
	}

	return 0, errors.New("user did not start the bot")
}

func (b *Bot) SendMessage(chatID int64, text string) (Message, error) {
	endpoint := TelegramAPI + b.Token + "/sendMessage"

	payload := SendMessagePayload{
		ChatID: chatID,
		Text:   text,
	}

	msgJson, err := json.Marshal(payload)
	if err != nil {
		return Message{}, fmt.Errorf("Marshal error: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(msgJson))
	if err != nil {
		return Message{}, fmt.Errorf("POST error: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := b.Client.Do(req)
	if err != nil {
		return Message{}, fmt.Errorf("Client Error")
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return Message{}, fmt.Errorf("Body Error")
	}

	var response Response[Message]

	if err := json.Unmarshal(body, &response); err != nil {
		return Message{}, fmt.Errorf("JSON decode error: %w", err)
	}

	if !response.OK {
		return Message{}, fmt.Errorf("Telegram error: %s", response.Description)
	}

	return response.Result, nil
}
