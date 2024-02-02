package main

import (
	"fmt"
	"io"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ChatUI struct {
	cr        *ChatRoom
	app       *tview.Application
	peersList *tview.TextView

	msgW    io.Writer
	inputCh chan string
	doneCh  chan struct{}
}

func NewChatUI(cr *ChatRoom) *ChatUI {
	app := tview.NewApplication()

	msgBox := tview.NewTextView()
	msgBox.SetDynamicColors(true)
	msgBox.SetBorder(true)
	msgBox.SetTitle(fmt.Sprintf("Room: %s", cr.roomName))

	msgBox.SetChangedFunc(func() {
		app.Draw()
	})

	inputCh := make(chan string, 32)
	input := tview.NewInputField().
		SetLabel(cr.nick + " > ").
		SetFieldWidth(0).
		SetFieldBackgroundColor(tcell.ColorBlack)

	input.SetDoneFunc(func(key tcell.Key) {
		if key != tcell.KeyEnter {
			return
		}
		line := input.GetText()
		if len(line) == 0 {
			return
		}

		if line == "/quit" {
			app.Stop()
			return
		}

		inputCh <- line
		input.SetText("")
	})

	peersList := tview.NewTextView()
	peersList.SetBorder(true)
	peersList.SetTitle("Peers")
	peersList.SetChangedFunc(func() { app.Draw() })

	chatPanel := tview.NewFlex().
		AddItem(msgBox, 0, 1, false).
		AddItem(peersList, 20, 1, false)

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(chatPanel, 0, 1, false).
		AddItem(input, 1, 1, true)

	app.SetRoot(flex, true)

	return &ChatUI{
		cr:        cr,
		app:       app,
		peersList: peersList,
		msgW:      msgBox,
		inputCh:   inputCh,
		doneCh:    make(chan struct{}, 1),
	}
}

func (ui *ChatUI) Run() error {
	go ui.handleEvents()
	defer ui.end()

	return ui.app.Run()
}

func (ui *ChatUI) end() {
	ui.doneCh <- struct{}{}
}

func (ui *ChatUI) refreshPeers() {
	peers := ui.cr.ListPeers()

	// clear is thread-safe
	ui.peersList.Clear()

	for _, p := range peers {
		fmt.Fprintln(ui.peersList, shortID(p))
	}

	ui.app.Draw()
}

func (ui *ChatUI) displayChatMessage(cm *ChatMessage) {
	prompt := withColor("green", fmt.Sprintf("<%s>:", cm.SenderNick))
	fmt.Fprintf(ui.msgW, "%s %s\n", prompt, cm.Message)
}

func (ui *ChatUI) displaySelfMessage(msg string) {
	prompt := withColor("yellow", fmt.Sprintf("<%s>:", ui.cr.nick))
	fmt.Fprintf(ui.msgW, "%s %s\n", prompt, msg)
}

func (ui *ChatUI) handleEvents() {
	peerRefreshTicker := time.NewTicker(time.Second)
	defer peerRefreshTicker.Stop()

	for {
		select {
		case input := <-ui.inputCh:
			err := ui.cr.Publish(input)
			if err != nil {
				printErr("publish error: %s", err)
			}
			ui.displaySelfMessage(input)

		case m := <-ui.cr.Messages:
			ui.displayChatMessage(m)

		case <-peerRefreshTicker.C:
			ui.refreshPeers()

		case <-ui.cr.ctx.Done():
			return

		case <-ui.doneCh:
			return
		}
	}
}

func withColor(color, msg string) string {
	return fmt.Sprintf("[%s]%s[-]", color, msg)
}
