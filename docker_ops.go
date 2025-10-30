package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (app *App) refreshContainers() {
	if app.dockerClient == nil {
		return
	}
	
	containers, err := app.dockerClient.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		app.showError(fmt.Sprintf("Failed to list containers: %v", err))
		return
	}

	app.containerList.Clear()
	
	// Headers
	headers := []string{"ID", "Image", "Command", "Created", "Status", "Ports", "Names"}
	for i, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignCenter).
			SetSelectable(false)
		app.containerList.SetCell(0, i, cell)
	}

	// Container data
	for i, container := range containers {
		row := i + 1
		
		// Truncate ID to first 12 chars
		id := container.ID
		if len(id) > 12 {
			id = id[:12]
		}
		
		// Get first image name
		image := container.Image
		if strings.Contains(image, ":") {
			parts := strings.Split(image, ":")
			image = parts[0]
		}
		
		// Truncate command
		command := container.Command
		if len(command) > 30 {
			command = command[:27] + "..."
		}
		
		// Format created time
		created := time.Unix(container.Created, 0).Format("2006-01-02 15:04")
		
		// Status with color
		status := container.Status
		statusColor := tcell.ColorRed
		if container.State == "running" {
			statusColor = tcell.ColorGreen
		} else if container.State == "exited" {
			statusColor = tcell.ColorYellow
		}
		
		// Ports
		ports := ""
		for _, port := range container.Ports {
			if port.PublicPort != 0 {
				ports += fmt.Sprintf("%d:%d ", port.PublicPort, port.PrivatePort)
			}
		}
		if len(ports) > 20 {
			ports = ports[:17] + "..."
		}
		
		// Names
		names := ""
		if len(container.Names) > 0 {
			names = strings.TrimPrefix(container.Names[0], "/")
		}
		
		cells := []struct {
			text  string
			color tcell.Color
		}{
			{id, tcell.ColorWhite},
			{image, tcell.ColorBlue},
			{command, tcell.ColorWhite},
			{created, tcell.ColorWhite},
			{status, statusColor},
			{ports, tcell.ColorWhite},
			{names, tcell.ColorWhite},
		}
		
		for j, cell := range cells {
			tableCell := tview.NewTableCell(cell.text).
				SetTextColor(cell.color).
				SetAlign(tview.AlignLeft)
			app.containerList.SetCell(row, j, tableCell)
		}
	}
	
	app.containerList.SetTitle(fmt.Sprintf("Containers (%d)", len(containers)))
	app.app.Draw()
}

func (app *App) refreshImages() {
	images, err := app.dockerClient.ImageList(context.Background(), image.ListOptions{})
	if err != nil {
		app.showError(fmt.Sprintf("Failed to list images: %v", err))
		return
	}

	app.imageList.Clear()
	
	// Headers
	headers := []string{"ID", "Repository", "Tag", "Created", "Size"}
	for i, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignCenter).
			SetSelectable(false)
		app.imageList.SetCell(0, i, cell)
	}

	// Image data
	for i, img := range images {
		row := i + 1
		
		// Truncate ID
		id := img.ID
		if strings.HasPrefix(id, "sha256:") {
			id = id[7:19] // Remove sha256: prefix and truncate
		}
		
		// Parse repository and tag
		repository := "<none>"
		tag := "<none>"
		if len(img.RepoTags) > 0 {
			parts := strings.Split(img.RepoTags[0], ":")
			if len(parts) >= 2 {
				repository = parts[0]
				tag = parts[1]
			}
		}
		
		// Format created time
		created := time.Unix(img.Created, 0).Format("2006-01-02")
		
		// Format size
		size := formatBytes(img.Size)
		
		cells := []string{id, repository, tag, created, size}
		
		for j, text := range cells {
			color := tcell.ColorWhite
			if j == 1 { // Repository
				color = tcell.ColorBlue
			} else if j == 2 { // Tag
				color = tcell.ColorGreen
			}
			
			cell := tview.NewTableCell(text).
				SetTextColor(color).
				SetAlign(tview.AlignLeft)
			app.imageList.SetCell(row, j, cell)
		}
	}
	
	app.imageList.SetTitle(fmt.Sprintf("Images (%d)", len(images)))
	app.app.Draw()
}

func (app *App) startContainer(containerID string) {
	err := app.dockerClient.ContainerStart(context.Background(), containerID, container.StartOptions{})
	if err != nil {
		app.showError(fmt.Sprintf("Failed to start container: %v", err))
		return
	}
	app.showSuccess(fmt.Sprintf("Started container %s", containerID[:12]))
	app.refreshContainers()
}

func (app *App) stopContainer(containerID string) {
	timeout := 10
	err := app.dockerClient.ContainerStop(context.Background(), containerID, container.StopOptions{
		Timeout: &timeout,
	})
	if err != nil {
		app.showError(fmt.Sprintf("Failed to stop container: %v", err))
		return
	}
	app.showSuccess(fmt.Sprintf("Stopped container %s", containerID[:12]))
	app.refreshContainers()
}

func (app *App) restartContainer(containerID string) {
	timeout := 10
	err := app.dockerClient.ContainerRestart(context.Background(), containerID, container.StopOptions{
		Timeout: &timeout,
	})
	if err != nil {
		app.showError(fmt.Sprintf("Failed to restart container: %v", err))
		return
	}
	app.showSuccess(fmt.Sprintf("Restarted container %s", containerID[:12]))
	app.refreshContainers()
}

func (app *App) removeContainer(containerID string) {
	// Show confirmation dialog
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Remove container %s?", containerID[:12])).
		AddButtons([]string{"Yes", "No"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			app.pages.RemovePage("modal")
			if buttonLabel == "Yes" {
				err := app.dockerClient.ContainerRemove(context.Background(), containerID, container.RemoveOptions{
					Force: true,
				})
				if err != nil {
					app.showError(fmt.Sprintf("Failed to remove container: %v", err))
					return
				}
				app.showSuccess(fmt.Sprintf("Removed container %s", containerID[:12]))
				app.refreshContainers()
			}
		})
	
	app.pages.AddPage("modal", modal, false, true)
}

func (app *App) removeImage(imageID string) {
	// Show confirmation dialog
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Remove image %s?", imageID[:12])).
		AddButtons([]string{"Yes", "No"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			app.pages.RemovePage("modal")
			if buttonLabel == "Yes" {
				_, err := app.dockerClient.ImageRemove(context.Background(), imageID, image.RemoveOptions{
					Force: true,
				})
				if err != nil {
					app.showError(fmt.Sprintf("Failed to remove image: %v", err))
					return
				}
				app.showSuccess(fmt.Sprintf("Removed image %s", imageID[:12]))
				app.refreshImages()
			}
		})
	
	app.pages.AddPage("modal", modal, false, true)
}

func (app *App) showLogs(containerID string) {
	app.logView.Clear()
	app.logView.SetTitle(fmt.Sprintf("Logs - %s", containerID[:12]))
	
	// Get logs
	logs, err := app.dockerClient.ContainerLogs(context.Background(), containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       "100",
		Follow:     true,
		Timestamps: true,
	})
	if err != nil {
		app.showError(fmt.Sprintf("Failed to get logs: %v", err))
		return
	}
	
	go func() {
		defer logs.Close()
		buf := make([]byte, 1024)
		for {
			n, err := logs.Read(buf)
			if err != nil {
				if err != io.EOF {
					app.showError(fmt.Sprintf("Error reading logs: %v", err))
				}
				break
			}
			
			app.app.QueueUpdateDraw(func() {
				fmt.Fprint(app.logView, string(buf[:n]))
			})
		}
	}()
	
	app.switchToLogs()
}

func (app *App) execIntoContainer(containerID string) {
	app.execIntoContainerWithShell(containerID)
}

func (app *App) showContainerDetails(containerID string) {
	inspect, err := app.dockerClient.ContainerInspect(context.Background(), containerID)
	if err != nil {
		app.showError(fmt.Sprintf("Failed to inspect container: %v", err))
		return
	}
	
	details := fmt.Sprintf(`[yellow]Container Details[white]

ID: %s
Name: %s
Image: %s
State: %s
Status: %s
Created: %s
Started: %s

[yellow]Network Settings[white]
IP Address: %s
Gateway: %s

[yellow]Mounts[white]`,
		inspect.ID,
		inspect.Name,
		inspect.Config.Image,
		inspect.State.Status,
		inspect.State.Status,
		inspect.Created,
		inspect.State.StartedAt,
		inspect.NetworkSettings.IPAddress,
		inspect.NetworkSettings.Gateway,
	)
	
	for _, mount := range inspect.Mounts {
		details += fmt.Sprintf("\n%s -> %s (%s)", mount.Source, mount.Destination, mount.Type)
	}
	
	// Environment variables
	if len(inspect.Config.Env) > 0 {
		details += "\n\n[yellow]Environment Variables[white]\n"
		for _, env := range inspect.Config.Env {
			details += fmt.Sprintf("%s\n", env)
		}
	}
	
	app.infoPanel.SetText(details)
	app.app.Draw()
}

func (app *App) showImageDetails(imageID string) {
	inspect, _, err := app.dockerClient.ImageInspectWithRaw(context.Background(), imageID)
	if err != nil {
		app.showError(fmt.Sprintf("Failed to inspect image: %v", err))
		return
	}
	
	details := fmt.Sprintf(`[yellow]Image Details[white]

ID: %s
Created: %s
Size: %s
Architecture: %s
OS: %s

[yellow]Tags[white]`,
		inspect.ID,
		inspect.Created,
		formatBytes(inspect.Size),
		inspect.Architecture,
		inspect.Os,
	)
	
	for _, tag := range inspect.RepoTags {
		details += fmt.Sprintf("\n%s", tag)
	}
	
	// Environment variables
	if len(inspect.Config.Env) > 0 {
		details += "\n\n[yellow]Environment Variables[white]\n"
		for _, env := range inspect.Config.Env {
			details += fmt.Sprintf("%s\n", env)
		}
	}
	
	// Command
	if len(inspect.Config.Cmd) > 0 {
		details += "\n\n[yellow]Default Command[white]\n"
		details += strings.Join(inspect.Config.Cmd, " ")
	}
	
	app.infoPanel.SetText(details)
	app.app.Draw()
}

func (app *App) showError(message string) {
	modal := tview.NewModal().
		SetText(message).
		SetBackgroundColor(tcell.ColorRed).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			app.pages.RemovePage("error")
		})
	
	app.pages.AddPage("error", modal, false, true)
}

func (app *App) showSuccess(message string) {
	modal := tview.NewModal().
		SetText(message).
		SetBackgroundColor(tcell.ColorGreen).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			app.pages.RemovePage("success")
		})
	
	app.pages.AddPage("success", modal, false, true)
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatDuration(d time.Duration) string {
	if d.Hours() >= 24 {
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd", days)
	} else if d.Hours() >= 1 {
		return fmt.Sprintf("%.0fh", d.Hours())
	} else if d.Minutes() >= 1 {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
	return fmt.Sprintf("%.0fs", d.Seconds())
}
