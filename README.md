<p align="center">
  <img src="assets/logo.png" alt="Docker Tui logo" width="180">
</p>

# Docker Management Tool

A command-line Docker management tool similar to k9s for Kubernetes. This tool provides a menu-driven interface for managing Docker containers, images, and viewing logs.

## Note on TUI Implementation

This project includes both a full TUI version (`main.go`) and a menu-driven CLI version (`docker-menu.go`). The TUI version may have compatibility issues with some terminal environments like Warp AI Terminal. The menu-driven version works reliably across all terminal environments.

## Features

- **Container Management**: List, start, stop, restart, and remove containers
- **Image Management**: List and remove Docker images
- **Log Viewing**: View and tail container logs in real-time
- **Shell Access**: Execute interactive shell sessions in containers
- **Resource Details**: View detailed information about containers and images

## Prerequisites

- Docker must be installed and running
- Go 1.19 or later

## Installation

```bash
git clone <repository>
cd docker-tui
go build -o docker-tui
```

## Usage

### TUI Version (Full Screen)
```bash
./bin/docker-tui
```

### Menu-Driven CLI Version (Recommended for Warp)
```bash
./bin/docker-menu
```

The menu-driven version provides all the same functionality through numbered menus and is more reliable across different terminal environments.

## Key Bindings

### Global
- `F1`: Switch to Containers view
- `F2`: Switch to Images view  
- `F3`: Switch to Logs view
- `q` or `Q`: Quit application
- `r` or `R`: Refresh current view

### Container View
- `↑/↓`: Navigate through containers
- `Enter`: Show container details
- `s`: Start selected container
- `t`: Stop selected container
- `x`: Restart selected container
- `l`: Show logs for selected container
- `e`: Execute shell in selected container
- `d`: Remove selected container (with confirmation)

### Image View
- `↑/↓`: Navigate through images
- `Enter`: Show image details
- `d`: Remove selected image (with confirmation)

### Log View
- Logs are displayed in real-time with tailing
- Use `F1` or `F2` to switch back to other views

## Interface Layout

```
┌─────────────────────────────────────────────────────────────┐
│          Docker TUI - F1: Containers | F2: Images | q: Quit │
├─────────────────────────────┬───────────────────────────────┤
│                             │                               │
│       Containers/Images     │         Details Panel         │
│          (Table)            │      (Container/Image info)   │
│                             │                               │
│                             ├───────────────────────────────┤
│                             │                               │
│                             │         Log Panel             │
│                             │     (Real-time logs)          │
│                             │                               │
└─────────────────────────────┴───────────────────────────────┘
```

## Container States

- **Running**: Green text
- **Exited**: Yellow text  
- **Other states**: Red text

## Notes

- The tool requires Docker daemon to be running
- Shell access attempts `/bin/sh` first, falls back to `/bin/bash` if needed
- Log tailing follows the last 100 lines by default
- All destructive operations (remove container/image) require confirmation
