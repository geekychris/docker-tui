package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
)

func main() {
	// Initialize Docker client
	cli, err := client.NewClientWithOpts(
		client.WithHost(fmt.Sprintf("unix://%s/.rd/docker.sock", os.Getenv("HOME"))),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	defer cli.Close()

	// Test Docker connection
	_, err = cli.Ping(context.Background())
	if err != nil {
		log.Fatalf("Failed to connect to Docker: %v", err)
	}

	fmt.Println("🐳 Docker CLI Tool")
	fmt.Println("================")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Println("Commands:")
		fmt.Println("  1. List containers")
		fmt.Println("  2. List images")
		fmt.Println("  3. Container operations")
		fmt.Println("  4. View logs")
		fmt.Println("  5. Exec into container")
		fmt.Println("  q. Quit")
		fmt.Print("\nEnter command: ")

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())

		switch input {
		case "1":
			listContainers(cli)
		case "2":
			listImages(cli)
		case "3":
			containerOperations(cli, scanner)
		case "4":
			viewLogs(cli, scanner)
		case "5":
			execIntoContainer(cli, scanner)
		case "q", "Q":
			fmt.Println("Goodbye!")
			return
		default:
			fmt.Println("Invalid command. Please try again.")
		}
		fmt.Println()
	}
}

func listContainers(cli *client.Client) {
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		fmt.Printf("Error listing containers: %v\n", err)
		return
	}

	fmt.Printf("\n📦 Found %d containers:\n", len(containers))
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("%-12s %-20s %-15s %-10s %s\n", "ID", "NAME", "IMAGE", "STATUS", "PORTS")
	fmt.Println(strings.Repeat("-", 80))

	for i, container := range containers {
		if i >= 20 { // Limit to 20 for readability
			fmt.Println("... and more")
			break
		}

		id := container.ID[:12]
		name := "unnamed"
		if len(container.Names) > 0 {
			name = strings.TrimPrefix(container.Names[0], "/")
		}

		image := container.Image
		if len(image) > 20 {
			image = image[:17] + "..."
		}

		status := container.State
		if status == "running" {
			status = "🟢 " + status
		} else {
			status = "🔴 " + status
		}

		ports := ""
		if len(container.Ports) > 0 {
			for j, port := range container.Ports {
				if j >= 2 { // Limit port display
					ports += "..."
					break
				}
				if port.PublicPort != 0 {
					ports += fmt.Sprintf("%d:%d ", port.PublicPort, port.PrivatePort)
				}
			}
		}

		fmt.Printf("%-12s %-20s %-15s %-15s %s\n", id, name, image, status, ports)
	}
}

func listImages(cli *client.Client) {
	images, err := cli.ImageList(context.Background(), image.ListOptions{})
	if err != nil {
		fmt.Printf("Error listing images: %v\n", err)
		return
	}

	fmt.Printf("\n🖼️  Found %d images:\n", len(images))
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("%-15s %-30s %-15s %s\n", "ID", "REPOSITORY", "TAG", "SIZE")
	fmt.Println(strings.Repeat("-", 80))

	for i, img := range images {
		if i >= 20 { // Limit to 20 for readability
			fmt.Println("... and more")
			break
		}

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

		if len(repository) > 30 {
			repository = repository[:27] + "..."
		}

		size := formatBytes(img.Size)

		fmt.Printf("%-15s %-30s %-15s %s\n", id, repository, tag, size)
	}
}

func containerOperations(cli *client.Client, scanner *bufio.Scanner) {
	fmt.Print("Enter container ID (first 12 chars): ")
	if !scanner.Scan() {
		return
	}
	containerID := strings.TrimSpace(scanner.Text())

	fmt.Println("Operations:")
	fmt.Println("  1. Start")
	fmt.Println("  2. Stop") 
	fmt.Println("  3. Restart")
	fmt.Println("  4. Remove")
	fmt.Print("Enter operation: ")

	if !scanner.Scan() {
		return
	}

	operation := strings.TrimSpace(scanner.Text())

	switch operation {
	case "1":
		err := cli.ContainerStart(context.Background(), containerID, container.StartOptions{})
		if err != nil {
			fmt.Printf("Error starting container: %v\n", err)
		} else {
			fmt.Printf("✅ Started container %s\n", containerID)
		}
	case "2":
		timeout := 10
		err := cli.ContainerStop(context.Background(), containerID, container.StopOptions{Timeout: &timeout})
		if err != nil {
			fmt.Printf("Error stopping container: %v\n", err)
		} else {
			fmt.Printf("✅ Stopped container %s\n", containerID)
		}
	case "3":
		timeout := 10
		err := cli.ContainerRestart(context.Background(), containerID, container.StopOptions{Timeout: &timeout})
		if err != nil {
			fmt.Printf("Error restarting container: %v\n", err)
		} else {
			fmt.Printf("✅ Restarted container %s\n", containerID)
		}
	case "4":
		fmt.Print("Are you sure you want to remove this container? (y/N): ")
		if scanner.Scan() {
			confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))
			if confirm == "y" || confirm == "yes" {
				err := cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
				if err != nil {
					fmt.Printf("Error removing container: %v\n", err)
				} else {
					fmt.Printf("✅ Removed container %s\n", containerID)
				}
			}
		}
	default:
		fmt.Println("Invalid operation.")
	}
}

func viewLogs(cli *client.Client, scanner *bufio.Scanner) {
	fmt.Print("Enter container ID: ")
	if !scanner.Scan() {
		return
	}
	containerID := strings.TrimSpace(scanner.Text())

	fmt.Print("Number of lines to show (default 50): ")
	scanner.Scan()
	linesStr := strings.TrimSpace(scanner.Text())
	lines := "50"
	if linesStr != "" {
		if _, err := strconv.Atoi(linesStr); err == nil {
			lines = linesStr
		}
	}

	logs, err := cli.ContainerLogs(context.Background(), containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       lines,
		Timestamps: true,
	})
	if err != nil {
		fmt.Printf("Error getting logs: %v\n", err)
		return
	}
	defer logs.Close()

	fmt.Printf("\n📋 Logs for container %s:\n", containerID)
	fmt.Println(strings.Repeat("-", 80))

	buf := make([]byte, 1024)
	for {
		n, err := logs.Read(buf)
		if err != nil {
			break
		}
		fmt.Print(string(buf[:n]))
	}
}

func execIntoContainer(cli *client.Client, scanner *bufio.Scanner) {
	fmt.Print("Enter container ID: ")
	if !scanner.Scan() {
		return
	}
	containerID := strings.TrimSpace(scanner.Text())

	fmt.Printf("Executing shell in container %s...\n", containerID)
	fmt.Println("Use 'exit' to return to this tool.")

	// Try different shells
	shells := []string{"/bin/bash", "/bin/sh", "/bin/zsh"}
	for _, shell := range shells {
		cmd := fmt.Sprintf("docker exec -it %s %s", containerID, shell)
		fmt.Printf("Trying: %s\n", cmd)
		
		// This would need to be handled differently in a real implementation
		// For now, just show the command
		fmt.Printf("Run this command in another terminal: %s\n", cmd)
		break
	}
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
