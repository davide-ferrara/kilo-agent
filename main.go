package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func readSystemPrompt() (string, error) {
	data, err := os.ReadFile("SYSTEM.md")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func main() {
	logFile, err := os.OpenFile("/tmp/kilo-agent.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags)

	// Composition root: wire the two actors (agent and TUI) through channels.
	events := make(chan Message)       // keyboard events TUI -> loop
	reqChan := make(chan string, 1)    // prompts TUI -> agent
	respChan := make(chan Message, 64) // model output agent -> loop

	systemPrompt, err := readSystemPrompt()
	if err != nil {
		panic(err)
	}

	tui := NewTUI(events, reqChan)
	tui.ModelName = "qwen3:14b"
	tui.Init()
	defer tui.Restore()

	agent := NewAgent("Kilo Agent", systemPrompt)
	go func() {
		defer func() {
			if value := recover(); value != nil {
				log.Printf("agent panic: %v", value)
				events <- Message{MsgType: MsgQuit}
			}
		}()
		agent.Run(reqChan, respChan)
	}()

	go tui.HandleInput()
	tui.Render()
	spinner := time.NewTicker(100 * time.Millisecond)
	defer spinner.Stop()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(signals)

	for {
		select {
		case msg := <-events:
			if msg.MsgType == MsgQuit {
				return
			}
			tui.Update(msg)
			tui.Render()
		case msg := <-respChan:
			tui.Update(msg)
			tui.Render()
		case <-spinner.C:
			if tui.Model.PendingRequests > 0 {
				tui.Update(Message{MsgType: MsgSpinnerTick})
				tui.Render()
			}
		case <-signals:
			return
		}
	}
}
