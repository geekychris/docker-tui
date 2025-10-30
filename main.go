package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/docker/docker/client"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type App struct {
	app           *tview.Application
	dockerClient  *client.Client
	pages         *tview.Pages
	containerList *tview.Table
	imageList     *tview.Table
	logView       *tview.TextView
	infoPanel     *tview.TextView
	currentView   string
}

func main() {
	app := &App{}
	
	// Initialize Docker client with multiple fallback options
	var err error
	app.dockerClient, err = app.createDockerClient()
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	defer app.dockerClient.Close()

	// Test Docker connection
	_, err = app.dockerClient.Ping(context.Background())
	if err != nil {
		log.Fatalf("Failed to connect to Docker: %v\nMake sure Docker is running and accessible.", err)
	}

	// Initialize and run TUI
	app.initUI()
	app.run()
}

func (app *App) initUI() {
	app.app = tview.NewApplication()
	app.pages = tview.NewPages()
	
	// Create main layout
	app.createMainLayout()
	
	// Set up global key bindings
	app.app.SetInputCapture(app.globalKeyHandler)
	
	app.app.SetRoot(app.pages, true)
}

func (app *App) createMainLayout() {
	// Create main flex layout
	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow)
	
	// Header
	header := tview.NewTextView().
		SetText("Docker TUI - F1: Containers | F2: Images | F3: Logs | q: Quit").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(tcell.ColorWhite).
		SetBackgroundColor(tcell.ColorBlue)
	header.SetBorder(true)
	
	// Content area
	contentFlex := tview.NewFlex()
	
	// Left panel - resource lists
	leftPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	
	// Container list
	app.containerList = tview.NewTable().
		SetBorders(true).
		SetSelectable(true, false)
	app.containerList.SetTitle("Containers").SetBorder(true)
	
	// Image list
	app.imageList = tview.NewTable().
		SetBorders(true).
		SetSelectable(true, false)
	app.imageList.SetTitle("Images").SetBorder(true)
	
	// Right panel - details and logs
	rightPanel := tview.NewFlex().SetDirection(tview.FlexRow)
	
	// Info panel
	app.infoPanel = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	app.infoPanel.SetTitle("Details").SetBorder(true)
	
	// Log view
	app.logView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetChangedFunc(func() {
			app.app.Draw()
		})
	app.logView.SetTitle("Logs").SetBorder(true)
	
	// Assemble layout
	leftPanel.AddItem(app.containerList, 0, 1, true)
	rightPanel.AddItem(app.infoPanel, 0, 1, false).
		AddItem(app.logView, 0, 1, false)
	
	contentFlex.AddItem(leftPanel, 0, 1, true).
		AddItem(rightPanel, 0, 1, false)
	
	mainFlex.AddItem(header, 3, 0, false).
		AddItem(contentFlex, 0, 1, true)
	
	app.pages.AddPage("main", mainFlex, true, true)
	
	// Set initial focus and load data
	app.currentView = "containers"
	app.app.SetFocus(app.containerList)
	
	// Set up container list key bindings
	app.containerList.SetInputCapture(app.containerKeyHandler)
	app.imageList.SetInputCapture(app.imageKeyHandler)
}

func (app *App) globalKeyHandler(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyF1:
		app.switchToContainers()
		return nil
	case tcell.KeyF2:
		app.switchToImages()
		return nil
	case tcell.KeyF3:
		app.switchToLogs()
		return nil
	case tcell.KeyRune:
		switch event.Rune() {
		case 'q', 'Q':
			app.app.Stop()
			return nil
		case 'r', 'R':
			app.refresh()
			return nil
		case 'h', 'H', '?':
			app.showHelp()
			return nil
		}
	}
	return event
}

func (app *App) containerKeyHandler(event *tcell.EventKey) *tcell.EventKey {
	row, _ := app.containerList.GetSelection()
	if row == 0 { // Header row
		return event
	}
	
	containerID := app.containerList.GetCell(row, 0).Text
	
	switch event.Key() {
	case tcell.KeyRune:
		switch event.Rune() {
		case 's':
			app.startContainer(containerID)
			return nil
		case 't':
			app.stopContainer(containerID)
			return nil
		case 'x':
			app.restartContainer(containerID)
			return nil
		case 'l':
			app.showLogs(containerID)
			return nil
		case 'e':
			app.execIntoContainer(containerID)
			return nil
		case 'd':
			app.removeContainer(containerID)
			return nil
		}
	case tcell.KeyEnter:
		app.showContainerDetails(containerID)
		return nil
	}
	return event
}

func (app *App) imageKeyHandler(event *tcell.EventKey) *tcell.EventKey {
	row, _ := app.imageList.GetSelection()
	if row == 0 { // Header row
		return event
	}
	
	imageID := app.imageList.GetCell(row, 0).Text
	
	switch event.Key() {
	case tcell.KeyRune:
		switch event.Rune() {
		case 'd':
			app.removeImage(imageID)
			return nil
		}
	case tcell.KeyEnter:
		app.showImageDetails(imageID)
		return nil
	}
	return event
}

func (app *App) switchToContainers() {
	app.currentView = "containers"
	app.app.SetFocus(app.containerList)
	app.refreshContainers()
}

func (app *App) switchToImages() {
	app.currentView = "images"
	app.app.SetFocus(app.imageList)
	app.refreshImages()
}

func (app *App) switchToLogs() {
	app.currentView = "logs"
	app.app.SetFocus(app.logView)
}

func (app *App) refresh() {
	switch app.currentView {
	case "containers":
		app.refreshContainers()
	case "images":
		app.refreshImages()
	}
}

func (app *App) createDockerClient() (*client.Client, error) {
	// Try to create client from environment first
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err == nil {
		// Test the connection
		_, pingErr := cli.Ping(context.Background())
		if pingErr == nil {
			return cli, nil
		}
		cli.Close()
	}

	// Try common Docker socket paths
	socketPaths := []string{
		"unix:///var/run/docker.sock",
		fmt.Sprintf("unix://%s/.rd/docker.sock", os.Getenv("HOME")),
		fmt.Sprintf("unix://%s/.docker/desktop/docker.sock", os.Getenv("HOME")),
		fmt.Sprintf("unix://%s/.docker/run/docker.sock", os.Getenv("HOME")),
	}

	for _, socketPath := range socketPaths {
		cli, err := client.NewClientWithOpts(
			client.WithHost(socketPath),
			client.WithAPIVersionNegotiation(),
		)
		if err != nil {
			continue
		}

		// Test the connection
		_, err = cli.Ping(context.Background())
		if err == nil {
			return cli, nil
		}
		cli.Close()
	}

	return nil, fmt.Errorf("could not connect to Docker daemon. Tried multiple socket paths")
}

func (app *App) run() {
	// Start the application
	if err := app.app.Run(); err != nil {
		log.Fatalf("Error running application: %v", err)
	}
}
