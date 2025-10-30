package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (app *App) showHelp() {
	helpText := `[yellow]Docker TUI - Help[white]

[yellow]Global Shortcuts:[white]
  F1          - Switch to Containers view
  F2          - Switch to Images view
  F3          - Switch to Logs view
  q/Q         - Quit application
  r/R         - Refresh current view
  h/?         - Show this help

[yellow]Container Operations:[white]
  ↑/↓         - Navigate through containers
  Enter       - Show container details
  s           - Start selected container
  t           - Stop selected container
  x           - Restart selected container
  l           - Show logs for selected container
  e           - Execute shell in selected container
  d           - Remove selected container (with confirmation)

[yellow]Image Operations:[white]
  ↑/↓         - Navigate through images
  Enter       - Show image details
  d           - Remove selected image (with confirmation)

[yellow]Log View:[white]
  Logs are displayed in real-time with tailing
  Use F1/F2 to switch back to other views

[yellow]Container States:[white]
  Running     - Green text
  Exited      - Yellow text
  Other       - Red text

Press ESC or q to close this help.`

	help := tview.NewTextView().
		SetText(helpText).
		SetDynamicColors(true).
		SetScrollable(true)
	help.SetBorder(true).SetTitle("Help")

	help.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || (event.Key() == tcell.KeyRune && (event.Rune() == 'q' || event.Rune() == 'Q')) {
			app.pages.RemovePage("help")
		}
		return event
	})

	app.pages.AddPage("help", help, true, true)
}

func (app *App) execIntoContainerWithShell(containerID string) {
	shells := []string{"/bin/bash", "/bin/sh", "/bin/zsh", "/bin/ash"}
	
	app.app.Suspend(func() {
		for _, shell := range shells {
			cmd := exec.Command("docker", "exec", "-it", containerID, shell)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			
			err := cmd.Run()
			if err == nil {
				return // Successfully executed
			}
		}
		
		// If all shells failed, show error
		fmt.Printf("Failed to exec into container with any shell\\n")
		fmt.Println("Press Enter to continue...")
		var input string
		fmt.Scanln(&input)
	})
}

func (app *App) confirmAction(message string, action func()) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"Yes", "No"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			app.pages.RemovePage("confirm")
			if buttonLabel == "Yes" {
				action()
			}
		})
	
	app.pages.AddPage("confirm", modal, false, true)
}
