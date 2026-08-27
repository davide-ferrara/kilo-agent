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
