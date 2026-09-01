package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"kilo-agent/telegram"
)

func RegisterTools() []Tool {
	tools := []Tool{}
	telegramIsBotConfiguredTool := Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "telegram_is_bot_configured",
			Description: "Check whether a Telegram bot token is configured for the agent.",
		},
	}

	tools = append(tools, telegramIsBotConfiguredTool)
	telegramStartPairingTool := Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "telegram_start_pairing",
			Description: "Create or return a Telegram pairing link. After calling this tool, show the link and end the response. Never call telegram_complete_pairing in the same turn.",
		},
	}
	telegramCompletePairingTool := Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "telegram_complete_pairing",
			Description: "Complete Telegram pairing only after a later user message explicitly confirms they opened the link and pressed Start. Never call this in the same turn as telegram_start_pairing.",
		},
	}
	telegramSendMessageTool := Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "telegram_send_message",
			Description: "Send a text message to the paired Telegram chat. If no chat is paired, start Telegram pairing first.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"text": {"type": "string", "description": "The text message to send"}
				},
				"required": ["text"]
			}`),
		},
	}

	tools = append(tools, telegramStartPairingTool, telegramCompletePairingTool, telegramSendMessageTool)

	randomIntTool := Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "random_int",
			Description: "Returns a random integer",
		},
	}

	randomIntNTool := Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "random_int_n",
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
			Name:        "read_file",
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
			Name:        "pwd",
			Description: "Returns the current working directory",
		},
	}

	tools = append(tools, pwdTool)

	lsTool := Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "ls",
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
			Name:        "write_file",
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
			Name:        "exec_cmd",
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

	webSearchTool := Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "web_search",
			Description: "Search the web or current news. Use news mode for recent events, headlines, and latest-news requests.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {"type": "string", "description": "The search query"},
					"limit": {"type": "integer", "description": "Maximum number of results (1-10, defaults to 8)"},
					"type": {"type": "string", "enum": ["web", "news"], "description": "Search type. Use news for recent events and latest news; defaults to web"}
				},
				"required": ["query"]
			}`),
		},
	}

	tools = append(tools, execCmdTool, webSearchTool)

	return tools
}

func (a *Agent) TelegramIsBotConfigured() string {
	if _, err := a.telegramBot(); err != nil {
		return "Telegram bot is not configured. Set telegram_bot_token in config.json."
	}
	return "Telegram bot is configured and ready to use."
}

func (a *Agent) TelegramStartPairing() string {
	if a.telegramStart != nil {
		return "A Telegram pairing is already pending. Show this link to the user: " +
			a.telegramStart.Url +
			". Stop now and wait for the user to confirm they pressed Start before calling telegram_complete_pairing."
	}

	bot, err := a.telegramBot()
	if err != nil {
		return "Telegram bot is not configured. Set telegram_bot_token in config.json."
	}

	start, err := bot.Start()
	if err != nil {
		return "Could not start Telegram pairing: " + err.Error()
	}
	a.telegramStart = &start

	return "Show this Telegram link to the user: " + start.Url +
		". Stop now. Do not call another pairing tool in this turn. Wait for a later user message confirming they pressed Start, then call telegram_complete_pairing."
}

func (a *Agent) TelegramCompletePairing() string {
	if a.telegramStart == nil {
		return "Telegram pairing has not been started. Call telegram_start_pairing first."
	}

	bot, err := a.telegramBot()
	if err != nil {
		return "Telegram bot is not configured. Set telegram_bot_token in config.json."
	}
	updates, err := bot.GetUpdates()
	if err != nil {
		return "Could not retrieve Telegram updates: " + err.Error()
	}
	chatID, err := telegram.GetChatID(*a.telegramStart, updates)
	if err != nil {
		return "Telegram pairing is still waiting for the user. Tell them to open the existing pairing link and press Start, then end the response. Do not call another pairing tool in this turn."
	}

	updatedConfig := a.Config
	updatedConfig.TelegramChatID = chatID
	if err := saveConfig(updatedConfig); err != nil {
		return "Found the Telegram chat, but could not save it: " + err.Error()
	}

	a.Config = updatedConfig
	a.telegramStart = nil
	return "Telegram chat paired successfully."
}

func (a *Agent) TelegramSendMessage(text string) string {
	bot, err := a.telegramBot()
	if err != nil {
		return "Telegram bot is not configured. Set telegram_bot_token in config.json."
	}
	if a.Config.TelegramChatID == 0 {
		return "Telegram chat is not paired. Call telegram_start_pairing first."
	}

	if _, err := bot.SendMessage(a.Config.TelegramChatID, text); err != nil {
		return "Could not send Telegram message: " + err.Error()
	}
	return "Telegram message sent successfully."
}

func RandomInt() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return strconv.Itoa(r.Int())
}

func RandomIntN(n int) string {
	if n <= 0 {
		return "panic"
	}
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
	if name == "rm" {
		return "Remove is forbidden for now."
	}

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

type BingRSS struct {
	Channel struct {
		Items []struct {
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			Description string `xml:"description"`
			PubDate     string `xml:"pubDate"`
		} `xml:"item"`
	} `xml:"channel"`
}

type WebPage struct {
	Title       string `json:"title"`
	Link        string `json:"url"`
	Description string `json:"snippet"`
	PubDate     string `json:"publication_date"`
}

const maxWebBodySize = 1 << 20 // 1 MiB
const maxWebResults = 10

// searchClient returns c when non-nil, otherwise a default client.
func searchClient(c *http.Client) *http.Client {
	if c != nil {
		return c
	}
	return &http.Client{Timeout: 10 * time.Second}
}

// WebSearch queries Microsoft Bing's RSS feed and returns the top `limit`
// matches. A non-positive limit defaults to 8. Empty queries are rejected.
func WebSearch(client *http.Client, query string, limit int, searchType string) ([]WebPage, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("web search: empty query")
	}
	if limit <= 0 {
		limit = 8
	} else if limit > maxWebResults {
		limit = maxWebResults
	}

	var path string
	switch searchType {
	case "", "web":
		path = "/search"
	case "news":
		path = "/news/search"
	default:
		return nil, fmt.Errorf("web search: invalid search type %q", searchType)
	}
	endpoint := "https://www.bing.com" + path + "?q=" + url.QueryEscape(query) + "&format=rss&adlt=strict"
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("web search: build request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:120.0) Gecko/20100101 Firefox/120.0")
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml")

	res, err := searchClient(client).Do(req)
	if err != nil {
		return nil, fmt.Errorf("web search: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return nil, fmt.Errorf("web search: bing returned %s: %s", res.Status, strings.TrimSpace(string(body)))
	}

	dec := xml.NewDecoder(io.LimitReader(res.Body, maxWebBodySize))
	var feed BingRSS
	if err := dec.Decode(&feed); err != nil {
		return nil, fmt.Errorf("web search: parse rss: %w", err)
	}

	if len(feed.Channel.Items) == 0 {
		return nil, fmt.Errorf("web search: no results for %q", query)
	}

	pages := make([]WebPage, 0, limit)
	for i, item := range feed.Channel.Items {
		if i >= limit {
			break
		}
		pages = append(pages, WebPage{
			Title:       item.Title,
			Link:        item.Link,
			Description: stripHTML(item.Description),
			PubDate:     item.PubDate,
		})
	}
	return pages, nil
}

// stripHTML removes all tags and unescapes entities in s.
func stripHTML(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return html.UnescapeString(b.String())
}

// WebSearchJSON marshals results as an indented JSON array for the model.
func WebSearchJSON(client *http.Client, query string, limit int, searchType string) (string, error) {
	pages, err := WebSearch(client, query, limit, searchType)
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(pages, "", "  ")
	if err != nil {
		return "", fmt.Errorf("web search: marshal results: %w", err)
	}
	return string(data), nil
}
