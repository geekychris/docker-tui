package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
)

type DockerTool struct {
	client *client.Client
	scanner *bufio.Scanner
}

func main() {
	tool := &DockerTool{
		scanner: bufio.NewScanner(os.Stdin),
	}

	// Initialize Docker client
	var err error
	tool.client, err = tool.createDockerClient()
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	defer tool.client.Close()

	// Test Docker connection
	_, err = tool.client.Ping(context.Background())
	if err != nil {
		log.Fatalf("Failed to connect to Docker: %v", err)
	}

	tool.showBanner()
	tool.mainMenu()
}

func (t *DockerTool) createDockerClient() (*client.Client, error) {
	// Try common Docker socket paths for different environments
	socketPaths := []string{
		fmt.Sprintf("unix://%s/.rd/docker.sock", os.Getenv("HOME")), // Rancher Desktop
		"unix:///var/run/docker.sock", // Standard Docker
		fmt.Sprintf("unix://%s/.docker/desktop/docker.sock", os.Getenv("HOME")), // Docker Desktop
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
			fmt.Printf("✅ Connected using: %s\n", socketPath)
			return cli, nil
		}
		cli.Close()
	}

	return nil, fmt.Errorf("could not connect to Docker daemon")
}

func (t *DockerTool) showBanner() {
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    🐳 Docker Management Tool                  ║")
	fmt.Println("║                        (k9s-style for Docker)                 ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

func (t *DockerTool) mainMenu() {
	for {
		fmt.Println("┌─ Main Menu ─────────────────────────────────────────────────┐")
		fmt.Println("│  1. 📦 Container Management                                 │")
		fmt.Println("│  2. 🖼️  Image Management                                    │")
		fmt.Println("│  3. 📋 View Logs                                           │")
		fmt.Println("│  4. 🔧 Quick Actions                                       │")
		fmt.Println("│  q. Quit                                                   │")
		fmt.Println("└─────────────────────────────────────────────────────────────┘")
		fmt.Print("\nSelect option: ")

		if !t.scanner.Scan() {
			break
		}

		input := strings.TrimSpace(t.scanner.Text())

		switch input {
		case "1":
			t.containerMenu()
		case "2":
			t.imageMenu()
		case "3":
			t.logsMenu()
		case "4":
			t.quickActionsMenu()
		case "q", "Q":
			fmt.Println("\n👋 Goodbye!")
			return
		default:
			fmt.Println("❌ Invalid option. Please try again.")
		}
		fmt.Println()
	}
}

func (t *DockerTool) containerMenu() {
	for {
		fmt.Println("\n┌─ Container Management ──────────────────────────────────────┐")
		fmt.Println("│  1. List all containers                                    │")
		fmt.Println("│  2. List running containers only                          │")
		fmt.Println("│  3. Start container                                       │")
		fmt.Println("│  4. Stop container                                        │")
		fmt.Println("│  5. Restart container                                     │")
		fmt.Println("│  6. Remove container                                      │")
		fmt.Println("│  7. Container details                                     │")
		fmt.Println("│  8. Execute shell in container                            │")
		fmt.Println("│  b. Back to main menu                                     │")
		fmt.Println("└─────────────────────────────────────────────────────────────┘")
		fmt.Print("\nSelect option: ")

		if !t.scanner.Scan() {
			break
		}

		input := strings.TrimSpace(t.scanner.Text())

		switch input {
		case "1":
			t.listContainers(true)
		case "2":
			t.listContainers(false)
		case "3":
			t.startContainer()
		case "4":
			t.stopContainer()
		case "5":
			t.restartContainer()
		case "6":
			t.removeContainer()
		case "7":
			t.containerDetails()
		case "8":
			t.execContainer()
		case "b", "B":
			return
		default:
			fmt.Println("❌ Invalid option.")
		}
	}
}

func (t *DockerTool) listContainers(includeAll bool) {
	containers, err := t.client.ContainerList(context.Background(), container.ListOptions{All: includeAll})
	if err != nil {
		fmt.Printf("❌ Error listing containers: %v\n", err)
		return
	}

	title := "Running Containers"
	if includeAll {
		title = "All Containers"
	}

	fmt.Printf("\n📦 %s (%d total):\n", title, len(containers))
	fmt.Println(strings.Repeat("═", 100))
	fmt.Printf("%-12s %-25s %-20s %-12s %-15s %s\n", "ID", "NAME", "IMAGE", "STATUS", "CREATED", "PORTS")
	fmt.Println(strings.Repeat("─", 100))

	for _, c := range containers {
		id := c.ID[:12]
		name := "unnamed"
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
			if len(name) > 25 {
				name = name[:22] + "..."
			}
		}

		img := c.Image
		if len(img) > 20 {
			img = img[:17] + "..."
		}

		status := c.State
		statusIcon := "🔴"
		if status == "running" {
			statusIcon = "🟢"
		} else if status == "exited" {
			statusIcon = "🟡"
		}

		created := time.Unix(c.Created, 0).Format("2006-01-02 15:04")

		ports := ""
		for i, port := range c.Ports {
			if i >= 2 {
				ports += "..."
				break
			}
			if port.PublicPort != 0 {
				ports += fmt.Sprintf("%d:%d ", port.PublicPort, port.PrivatePort)
			}
		}
		if len(ports) > 15 {
			ports = ports[:12] + "..."
		}

		fmt.Printf("%-12s %-25s %-20s %s%-11s %-15s %s\n", 
			id, name, img, statusIcon, status, created, ports)
	}
	fmt.Println(strings.Repeat("═", 100))
}

func (t *DockerTool) startContainer() {
	fmt.Print("Enter container ID or name: ")
	if !t.scanner.Scan() {
		return
	}
	containerID := strings.TrimSpace(t.scanner.Text())

	err := t.client.ContainerStart(context.Background(), containerID, container.StartOptions{})
	if err != nil {
		fmt.Printf("❌ Error starting container: %v\n", err)
	} else {
		fmt.Printf("✅ Started container %s\n", containerID)
	}
}

func (t *DockerTool) stopContainer() {
	fmt.Print("Enter container ID or name: ")
	if !t.scanner.Scan() {
		return
	}
	containerID := strings.TrimSpace(t.scanner.Text())

	timeout := 10
	err := t.client.ContainerStop(context.Background(), containerID, container.StopOptions{Timeout: &timeout})
	if err != nil {
		fmt.Printf("❌ Error stopping container: %v\n", err)
	} else {
		fmt.Printf("✅ Stopped container %s\n", containerID)
	}
}

func (t *DockerTool) restartContainer() {
	fmt.Print("Enter container ID or name: ")
	if !t.scanner.Scan() {
		return
	}
	containerID := strings.TrimSpace(t.scanner.Text())

	timeout := 10
	err := t.client.ContainerRestart(context.Background(), containerID, container.StopOptions{Timeout: &timeout})
	if err != nil {
		fmt.Printf("❌ Error restarting container: %v\n", err)
	} else {
		fmt.Printf("✅ Restarted container %s\n", containerID)
	}
}

func (t *DockerTool) removeContainer() {
	fmt.Print("Enter container ID or name: ")
	if !t.scanner.Scan() {
		return
	}
	containerID := strings.TrimSpace(t.scanner.Text())

	fmt.Printf("⚠️  Are you sure you want to remove container %s? (y/N): ", containerID)
	if !t.scanner.Scan() {
		return
	}
	confirm := strings.ToLower(strings.TrimSpace(t.scanner.Text()))

	if confirm == "y" || confirm == "yes" {
		err := t.client.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
		if err != nil {
			fmt.Printf("❌ Error removing container: %v\n", err)
		} else {
			fmt.Printf("✅ Removed container %s\n", containerID)
		}
	} else {
		fmt.Println("❌ Cancelled")
	}
}

func (t *DockerTool) containerDetails() {
	fmt.Print("Enter container ID or name: ")
	if !t.scanner.Scan() {
		return
	}
	containerID := strings.TrimSpace(t.scanner.Text())

	inspect, err := t.client.ContainerInspect(context.Background(), containerID)
	if err != nil {
		fmt.Printf("❌ Error inspecting container: %v\n", err)
		return
	}

	fmt.Printf("\n🔍 Container Details for %s:\n", containerID)
	fmt.Println(strings.Repeat("═", 60))
	fmt.Printf("Name: %s\n", inspect.Name)
	fmt.Printf("Image: %s\n", inspect.Config.Image)
	fmt.Printf("State: %s\n", inspect.State.Status)
	fmt.Printf("Created: %s\n", inspect.Created)
	fmt.Printf("Started: %s\n", inspect.State.StartedAt)
	fmt.Printf("IP Address: %s\n", inspect.NetworkSettings.IPAddress)
	
	if len(inspect.Mounts) > 0 {
		fmt.Println("\nMounts:")
		for _, mount := range inspect.Mounts {
			fmt.Printf("  %s -> %s (%s)\n", mount.Source, mount.Destination, mount.Type)
		}
	}
}

func (t *DockerTool) execContainer() {
	fmt.Print("Enter container ID or name: ")
	if !t.scanner.Scan() {
		return
	}
	containerID := strings.TrimSpace(t.scanner.Text())

	fmt.Printf("🚀 Executing shell in container %s...\n", containerID)
	fmt.Println("Use 'exit' to return to this tool.")

	// Try different shells
	shells := []string{"/bin/bash", "/bin/sh", "/bin/zsh", "/bin/ash"}
	for _, shell := range shells {
		fmt.Printf("Trying %s...\n", shell)
		cmd := exec.Command("docker", "exec", "-it", containerID, shell)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err == nil {
			fmt.Printf("Returned from container %s\n", containerID)
			return
		}
	}
	
	fmt.Println("❌ Failed to exec into container with any shell")
}

func (t *DockerTool) imageMenu() {
	for {
		fmt.Println("\n┌─ Image Management ──────────────────────────────────────────┐")
		fmt.Println("│  1. List all images                                        │")
		fmt.Println("│  2. Remove image                                           │")
		fmt.Println("│  3. Image details                                         │")
		fmt.Println("│  b. Back to main menu                                     │")
		fmt.Println("└─────────────────────────────────────────────────────────────┘")
		fmt.Print("\nSelect option: ")

		if !t.scanner.Scan() {
			break
		}

		input := strings.TrimSpace(t.scanner.Text())

		switch input {
		case "1":
			t.listImages()
		case "2":
			t.removeImage()
		case "3":
			t.imageDetails()
		case "b", "B":
			return
		default:
			fmt.Println("❌ Invalid option.")
		}
	}
}

func (t *DockerTool) listImages() {
	images, err := t.client.ImageList(context.Background(), image.ListOptions{})
	if err != nil {
		fmt.Printf("❌ Error listing images: %v\n", err)
		return
	}

	fmt.Printf("\n🖼️  Found %d images:\n", len(images))
	fmt.Println(strings.Repeat("═", 85))
	fmt.Printf("%-15s %-35s %-15s %-10s %s\n", "ID", "REPOSITORY", "TAG", "SIZE", "CREATED")
	fmt.Println(strings.Repeat("─", 85))

	for _, img := range images {
		id := img.ID
		if strings.HasPrefix(id, "sha256:") {
			id = id[7:19]
		}

		repository := "<none>"
		tag := "<none>"
		if len(img.RepoTags) > 0 {
			parts := strings.Split(img.RepoTags[0], ":")
			if len(parts) >= 2 {
				repository = parts[0]
				tag = parts[1]
			}
		}

		if len(repository) > 35 {
			repository = repository[:32] + "..."
		}

		size := formatBytes(img.Size)
		created := time.Unix(img.Created, 0).Format("2006-01-02")

		fmt.Printf("%-15s %-35s %-15s %-10s %s\n", id, repository, tag, size, created)
	}
	fmt.Println(strings.Repeat("═", 85))
}

func (t *DockerTool) removeImage() {
	fmt.Print("Enter image ID or name: ")
	if !t.scanner.Scan() {
		return
	}
	imageID := strings.TrimSpace(t.scanner.Text())

	fmt.Printf("⚠️  Are you sure you want to remove image %s? (y/N): ", imageID)
	if !t.scanner.Scan() {
		return
	}
	confirm := strings.ToLower(strings.TrimSpace(t.scanner.Text()))

	if confirm == "y" || confirm == "yes" {
		_, err := t.client.ImageRemove(context.Background(), imageID, image.RemoveOptions{Force: true})
		if err != nil {
			fmt.Printf("❌ Error removing image: %v\n", err)
		} else {
			fmt.Printf("✅ Removed image %s\n", imageID)
		}
	} else {
		fmt.Println("❌ Cancelled")
	}
}

func (t *DockerTool) imageDetails() {
	fmt.Print("Enter image ID or name: ")
	if !t.scanner.Scan() {
		return
	}
	imageID := strings.TrimSpace(t.scanner.Text())

	inspect, _, err := t.client.ImageInspectWithRaw(context.Background(), imageID)
	if err != nil {
		fmt.Printf("❌ Error inspecting image: %v\n", err)
		return
	}

	fmt.Printf("\n🔍 Image Details for %s:\n", imageID)
	fmt.Println(strings.Repeat("═", 60))
	fmt.Printf("ID: %s\n", inspect.ID)
	fmt.Printf("Created: %s\n", inspect.Created)
	fmt.Printf("Size: %s\n", formatBytes(inspect.Size))
	fmt.Printf("Architecture: %s\n", inspect.Architecture)
	fmt.Printf("OS: %s\n", inspect.Os)
	
	if len(inspect.RepoTags) > 0 {
		fmt.Println("\nTags:")
		for _, tag := range inspect.RepoTags {
			fmt.Printf("  %s\n", tag)
		}
	}
}

func (t *DockerTool) logsMenu() {
	fmt.Print("Enter container ID or name: ")
	if !t.scanner.Scan() {
		return
	}
	containerID := strings.TrimSpace(t.scanner.Text())

	fmt.Print("Number of lines to show (default 50): ")
	t.scanner.Scan()
	linesStr := strings.TrimSpace(t.scanner.Text())
	lines := "50"
	if linesStr != "" {
		if _, err := strconv.Atoi(linesStr); err == nil {
			lines = linesStr
		}
	}

	fmt.Print("Follow logs? (y/N): ")
	t.scanner.Scan()
	follow := strings.ToLower(strings.TrimSpace(t.scanner.Text())) == "y"

	logs, err := t.client.ContainerLogs(context.Background(), containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       lines,
		Follow:     follow,
		Timestamps: true,
	})
	if err != nil {
		fmt.Printf("❌ Error getting logs: %v\n", err)
		return
	}
	defer logs.Close()

	fmt.Printf("\n📋 Logs for container %s:\n", containerID)
	fmt.Println(strings.Repeat("═", 80))

	buf := make([]byte, 1024)
	for {
		n, err := logs.Read(buf)
		if err != nil {
			break
		}
		fmt.Print(string(buf[:n]))
	}
	
	if follow {
		fmt.Println("\n[Press Ctrl+C to stop following logs]")
	}
}

func (t *DockerTool) quickActionsMenu() {
	fmt.Println("\n┌─ Quick Actions ─────────────────────────────────────────────┐")
	fmt.Println("│  1. Stop all running containers                            │")
	fmt.Println("│  2. Remove all stopped containers                         │")
	fmt.Println("│  3. Show Docker system info                               │")
	fmt.Println("│  4. Clean up unused images                                │")
	fmt.Println("│  b. Back to main menu                                     │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")
	fmt.Print("\nSelect option: ")

	if !t.scanner.Scan() {
		return
	}

	input := strings.TrimSpace(t.scanner.Text())

	switch input {
	case "1":
		t.stopAllContainers()
	case "2":
		t.removeStoppedContainers()
	case "3":
		t.showSystemInfo()
	case "4":
		t.cleanupImages()
	case "b", "B":
		return
	default:
		fmt.Println("❌ Invalid option.")
	}
}

func (t *DockerTool) stopAllContainers() {
	containers, err := t.client.ContainerList(context.Background(), container.ListOptions{})
	if err != nil {
		fmt.Printf("❌ Error listing containers: %v\n", err)
		return
	}

	if len(containers) == 0 {
		fmt.Println("ℹ️  No running containers found.")
		return
	}

	fmt.Printf("⚠️  Stop %d running containers? (y/N): ", len(containers))
	if !t.scanner.Scan() {
		return
	}
	confirm := strings.ToLower(strings.TrimSpace(t.scanner.Text()))

	if confirm == "y" || confirm == "yes" {
		for _, c := range containers {
			timeout := 10
			err := t.client.ContainerStop(context.Background(), c.ID, container.StopOptions{Timeout: &timeout})
			if err != nil {
				fmt.Printf("❌ Failed to stop %s: %v\n", c.ID[:12], err)
			} else {
				fmt.Printf("✅ Stopped %s\n", c.ID[:12])
			}
		}
	}
}

func (t *DockerTool) removeStoppedContainers() {
	// Get all containers and filter manually
	allContainers, err := t.client.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		fmt.Printf("❌ Error listing containers: %v\n", err)
		return
	}
	
	// Filter for stopped containers
	var containers []types.Container
	for _, c := range allContainers {
		if c.State == "exited" {
			containers = append(containers, c)
		}
	}
	if err != nil {
		fmt.Printf("❌ Error listing containers: %v\n", err)
		return
	}

	if len(containers) == 0 {
		fmt.Println("ℹ️  No stopped containers found.")
		return
	}

	fmt.Printf("⚠️  Remove %d stopped containers? (y/N): ", len(containers))
	if !t.scanner.Scan() {
		return
	}
	confirm := strings.ToLower(strings.TrimSpace(t.scanner.Text()))

	if confirm == "y" || confirm == "yes" {
		for _, c := range containers {
			err := t.client.ContainerRemove(context.Background(), c.ID, container.RemoveOptions{})
			if err != nil {
				fmt.Printf("❌ Failed to remove %s: %v\n", c.ID[:12], err)
			} else {
				fmt.Printf("✅ Removed %s\n", c.ID[:12])
			}
		}
	}
}

func (t *DockerTool) showSystemInfo() {
	info, err := t.client.Info(context.Background())
	if err != nil {
		fmt.Printf("❌ Error getting system info: %v\n", err)
		return
	}

	fmt.Println("\n🖥️  Docker System Information:")
	fmt.Println(strings.Repeat("═", 40))
	fmt.Printf("Containers: %d (running: %d)\n", info.Containers, info.ContainersRunning)
	fmt.Printf("Images: %d\n", info.Images)
	fmt.Printf("Server Version: %s\n", info.ServerVersion)
	fmt.Printf("Storage Driver: %s\n", info.Driver)
	fmt.Printf("Operating System: %s\n", info.OperatingSystem)
	fmt.Printf("Architecture: %s\n", info.Architecture)
	fmt.Printf("CPUs: %d\n", info.NCPU)
	fmt.Printf("Total Memory: %s\n", formatBytes(info.MemTotal))
}

func (t *DockerTool) cleanupImages() {
	fmt.Println("🧹 This would remove dangling images...")
	fmt.Println("ℹ️  Run 'docker image prune' manually to clean up unused images.")
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
