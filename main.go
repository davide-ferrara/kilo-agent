package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kilo-agent/tui"
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
	events := make(chan tui.Message)       // keyboard events TUI -> loop
	reqChan := make(chan string, 16)       // prompts TUI -> agent
	respChan := make(chan tui.Message, 64) // model output agent -> loop

	systemPrompt, err := readSystemPrompt()
	if err != nil {
		panic(err)
	}

	tuiApp := tui.New(events)
	if err := tuiApp.Init("qwen3:14b"); err != nil {
		panic(err)
	}
	defer tuiApp.Restore()

	agent := NewAgent("Kilo Agent", systemPrompt)
	go agent.Run(reqChan, respChan)

	go tuiApp.HandleInput()
	tuiApp.Render()
	spinner := time.NewTicker(100 * time.Millisecond)
	defer spinner.Stop()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGWINCH)
	defer signal.Stop(signals)

	for {
		select {
		case msg := <-events:
			if dispatchEffects(tuiApp.Update(msg), reqChan) {
				return
			}
			tuiApp.Render()
		case msg := <-respChan:
			if dispatchEffects(tuiApp.Update(msg), reqChan) {
				return
			}
			tuiApp.Render()
		case <-spinner.C:
			if tuiApp.Pending() {
				tuiApp.Update(tui.Message{MsgType: tui.MsgSpinnerTick})
				tuiApp.Render()
			}
		case signal := <-signals:
			if signal == syscall.SIGWINCH {
				if sizeErr := tuiApp.Resize(); sizeErr == nil {
					tuiApp.Render()
				}
				continue
			}
			return
		}
	}
}

func dispatchEffects(effects []tui.Effect, requests chan<- string) bool {
	for _, effect := range effects {
		switch effect.Type {
		case tui.EffectSendPrompt:
			requests <- expandFileReferences(effect.Data)
		case tui.EffectQuit:
			return true
		}
	}
	return false
}
