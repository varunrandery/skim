# Repository Guidelines

## Overview

Skim is a terminal-based speed reading application written in Go. It uses the Bubble Tea TUI framework and displays text one word at a time with an Optimal Recognition Point (ORP) highlight.

## Project Structure

```
skim/
├── main.go          # Single-file application (all source code)
├── go.mod           # Go module definition
├── go.sum           # Dependency checksums
├── sample.txt       # Sample text file for testing
└── .gitignore       # Git ignore rules
```

This is a single-file Go project. All application logic resides in `main.go`.

## Build, Test, and Development Commands

| Command | Description |
|---------|-------------|
| `go build` | Compile the binary (outputs `skim`) |
| `go run main.go [file]` | Build and run directly |
| `go run main.go -wpm 300 file.txt` | Run with custom WPM setting |
| `go mod tidy` | Clean up dependencies |
| `go fmt ./...` | Format code |
| `go vet ./...` | Run static analysis |

### Running the Application

```bash
# Open file picker
go run main.go

# Open specific file
go run main.go sample.txt

# Set initial WPM (50-1000)
go run main.go -wpm 400 sample.txt
```

## Coding Style & Naming Conventions

- **Language**: Go 1.25+
- **Formatting**: Standard `gofmt` rules (tabs for indentation)
- **Naming**: camelCase for unexported, PascalCase for exported identifiers
- **Framework**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI architecture (Model-Update-View pattern)

### Key Patterns

- Model struct holds all application state
- `Update()` handles messages and returns commands
- `View()` renders the UI as a string
- Key bindings defined via `key.Binding` structs

## Testing Guidelines

No test suite currently exists. When adding tests:

- Name test files `*_test.go`
- Use standard Go testing: `go test ./...`
- Follow table-driven test patterns

## Commit & Pull Request Guidelines

### Commit Messages

Use short, descriptive commit messages in imperative mood:
- `Fix height calculation bug`
- `Add file picker support`
- `Update dependencies`

### Pull Requests

- Provide a clear description of changes
- Test the application manually before submitting
- Ensure code passes `go fmt` and `go vet`

## Architecture Notes

The application follows Bubble Tea's Elm-inspired architecture:

1. **Model**: Holds state (words, position, WPM, UI components)
2. **Update**: Processes key events and timer ticks
3. **View**: Renders current word with ORP highlighting

Key components:
- `filepicker`: File selection UI
- `progress`: Reading progress bar
- `help`: Keyboard shortcut display
