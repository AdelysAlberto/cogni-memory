package cli

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// SelectItem represents an option in the interactive selector
type SelectItem struct {
	Key         string
	Title       string
	Description string
}

// InteractiveSelect displays a windowed, filterable TUI list with arrow key navigation
func InteractiveSelect(prompt string, items []SelectItem, defaultIndex int, windowSize int) (SelectItem, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		// Non-interactive fallback (e.g. piped stdin)
		if defaultIndex >= 0 && defaultIndex < len(items) {
			return items[defaultIndex], nil
		}
		return items[0], nil
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		if defaultIndex >= 0 && defaultIndex < len(items) {
			return items[defaultIndex], nil
		}
		return items[0], nil
	}
	defer func() {
		_ = term.Restore(fd, oldState)
	}()

	if windowSize <= 0 {
		windowSize = 4
	}

	filter := ""
	selectedIndex := defaultIndex
	if selectedIndex < 0 || selectedIndex >= len(items) {
		selectedIndex = 0
	}

	// Filter items based on query
	getFiltered := func() []SelectItem {
		if filter == "" {
			return items
		}
		var result []SelectItem
		q := strings.ToLower(filter)
		for _, item := range items {
			if strings.Contains(strings.ToLower(item.Key), q) ||
				strings.Contains(strings.ToLower(item.Title), q) ||
				strings.Contains(strings.ToLower(item.Description), q) {
				result = append(result, item)
			}
		}
		return result
	}

	renderedLines := 0

	clearRendered := func() {
		for i := 0; i < renderedLines; i++ {
			// Move up 1 line and clear line
			fmt.Print("\033[1A\033[2K\r")
		}
		renderedLines = 0
	}

	render := func() {
		filtered := getFiltered()
		if len(filtered) == 0 {
			selectedIndex = 0
		} else if selectedIndex >= len(filtered) {
			selectedIndex = len(filtered) - 1
		} else if selectedIndex < 0 {
			selectedIndex = 0
		}

		var lines []string
		lines = append(lines, fmt.Sprintf("\r\033[1;36m%s\033[0m \033[90m(Filtra escribiendo, ↑/↓ para mover, Enter para elegir)\033[0m", prompt))

		filterDisplay := filter
		if filterDisplay == "" {
			filterDisplay = "\033[90m[Escribe para buscar...]\033[0m"
		} else {
			filterDisplay = fmt.Sprintf("\033[1;33m%s\033[0m", filter)
		}
		lines = append(lines, fmt.Sprintf("\r  🔍 Filtro: %s", filterDisplay))

		if len(filtered) == 0 {
			lines = append(lines, "\r  \033[31mNo se encontraron arneses con ese filtro.\033[0m")
		} else {
			// Windowing calculations
			start := 0
			if selectedIndex >= windowSize {
				start = selectedIndex - windowSize + 1
			}
			end := start + windowSize
			if end > len(filtered) {
				end = len(filtered)
				if end-windowSize >= 0 {
					start = end - windowSize
				} else {
					start = 0
				}
			}

			// Top indicator
			if start > 0 {
				lines = append(lines, fmt.Sprintf("\r  \033[90m▲ (%d más arriba)\033[0m", start))
			} else {
				lines = append(lines, "\r")
			}

			for i := start; i < end; i++ {
				item := filtered[i]
				if i == selectedIndex {
					lines = append(lines, fmt.Sprintf("\r  \033[1;32m❯ %s\033[0m \033[90m(%s)\033[0m", item.Title, item.Description))
				} else {
					lines = append(lines, fmt.Sprintf("\r    \033[37m%s\033[0m \033[90m(%s)\033[0m", item.Title, item.Description))
				}
			}

			// Bottom indicator
			if end < len(filtered) {
				lines = append(lines, fmt.Sprintf("\r  \033[90m▼ (%d más abajo)\033[0m", len(filtered)-end))
			} else {
				lines = append(lines, "\r")
			}
		}

		clearRendered()
		for _, l := range lines {
			fmt.Println(l)
		}
		renderedLines = len(lines)
	}

	render()

	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			break
		}

		filtered := getFiltered()

		if n == 1 {
			b := buf[0]
			switch b {
			case 13, 10: // Enter
				if len(filtered) > 0 {
					clearRendered()
					return filtered[selectedIndex], nil
				}
			case 3: // Ctrl+C
				clearRendered()
				return SelectItem{Key: "none", Title: "Cancelado"}, fmt.Errorf("interrumpido por el usuario")
			case 127, 8: // Backspace
				if len(filter) > 0 {
					filter = filter[:len(filter)-1]
					selectedIndex = 0
					render()
				}
			case 9: // Tab -> down
				if len(filtered) > 0 {
					selectedIndex = (selectedIndex + 1) % len(filtered)
					render()
				}
			default:
				if b >= 32 && b <= 126 { // Printable character
					filter += string(b)
					selectedIndex = 0
					render()
				}
			}
		} else if n == 3 && buf[0] == 27 && buf[1] == 91 { // ANSI Escape sequence (Arrows)
			switch buf[2] {
			case 65: // Up Arrow
				if len(filtered) > 0 {
					if selectedIndex > 0 {
						selectedIndex--
					} else {
						selectedIndex = len(filtered) - 1
					}
					render()
				}
			case 66: // Down Arrow
				if len(filtered) > 0 {
					if selectedIndex < len(filtered)-1 {
						selectedIndex++
					} else {
						selectedIndex = 0
					}
					render()
				}
			}
		}
	}

	clearRendered()
	if defaultIndex >= 0 && defaultIndex < len(items) {
		return items[defaultIndex], nil
	}
	return items[0], nil
}
