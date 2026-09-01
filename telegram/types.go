package telegram

import "net/http"

type Response[T any] struct {
	OK          bool   `json:"ok"`
	Result      T      `json:"result"`
	Description string `json:"description,omitempty"`
}

type Bot struct {
	Client *http.Client
	Token  string
}

type User struct {
	ID                         int64   `json:"id,omitempty"`
	IsBot                      bool    `json:"is_bot,omitempty"`
	FirstName                  string  `json:"first_name,omitempty"`
	LastName                   *string `json:"last_name,omitempty"`
	Username                   *string `json:"username,omitempty"`
	LanguageCode               *string `json:"language_code,omitempty"`
	IsPremium                  *bool   `json:"is_premium,omitempty"`
	AddedToAttachmentMenu      *bool   `json:"added_to_attachment_menu,omitempty"`
	CanJoinGroups              *bool   `json:"can_join_groups,omitempty"`
	CanReadAllGroupMessages    *bool   `json:"can_read_all_group_messages,omitempty"`
	SupportsGuestQueries       *bool   `json:"supports_guest_queries,omitempty"`
	SupportsInlineQueries      *bool   `json:"supports_inline_queries,omitempty"`
	CanConnectToBusiness       *bool   `json:"can_connect_to_business,omitempty"`
	HasMainWebApp              *bool   `json:"has_main_web_app,omitempty"`
	HasTopicsEnabled           *bool   `json:"has_topics_enabled,omitempty"`
	AllowsUsersToCreateTopics  *bool   `json:"allows_users_to_create_topics,omitempty"`
	CanManageBots              *bool   `json:"can_manage_bots,omitempty"`
	SupportsJoinRequestQueries *bool   `json:"supports_join_request_queries,omitempty"`
}

type Chat struct {
	ID               int64   `json:"id"`
	Type             string  `json:"type"`
	Title            *string `json:"title,omitempty"`
	Username         *string `json:"username,omitempty"`
	FirstName        *string `json:"first_name,omitempty"`
	LastName         *string `json:"last_name,omitempty"`
	IsForum          *bool   `json:"is_forum,omitempty"`
	IsDirectMessages *bool   `json:"is_direct_messages,omitempty"`
}

type SendMessagePayload struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

type Message struct {
	MessageId       int     `json:"message_id"`
	MessageThreadID int     `json:"message_thread_id"`
	From            *User   `json:"from,omitempty"`
	Date            int     `json:"date"`
	Chat            Chat    `json:"chat"`
	Text            *string `json:"text,omitempty"`
}

type Update struct {
	UpdateId int64   `json:"update_id"`
	Message  Message `json:"message"`
}

type Start struct {
	Url string
	Otp int
}
