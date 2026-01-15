package styles

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Color Palette
var (
	ColorPrimary   = lipgloss.Color("#bd93f9") // Dracula Purple
	ColorSecondary = lipgloss.Color("#ff79c6") // Dracula Pink
	ColorSuccess   = lipgloss.Color("#50fa7b") // Dracula Green
	ColorWarning   = lipgloss.Color("#ffb86c") // Dracula Orange
	ColorError     = lipgloss.Color("#ff5555") // Dracula Red
	ColorText      = lipgloss.Color("#f8f8f2") // Dracula Foreground
	ColorSubtext   = lipgloss.Color("#6272a4") // Dracula Comment
	ColorBg        = lipgloss.Color("#282a36") // Dracula Background
	ColorCyan      = lipgloss.Color("#8be9fd") // Dracula Cyan
)

// Styles
var (
	// App Container
	DocStyle = lipgloss.NewStyle().
			Margin(1, 2)

	// Titles
	TitleStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			MarginBottom(1)

	// Menu
	MenuTitleStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true).
			MarginBottom(1)

	MenuItemStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			PaddingLeft(2)

	MenuItemSelectedStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess).
				Bold(true).
				PaddingLeft(0).
				SetString("> ")

	// Stats
	StatKeyStyle = lipgloss.NewStyle().
			Foreground(ColorSubtext).
			Width(32)

	StatValueStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Bold(true)

	StatBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSubtext).
			Padding(1, 2).
			Width(88).
			Height(10)

	// Help/Footer
	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorSubtext).
			MarginTop(2).
			Italic(true)

	// Table
	TableHeaderStyle = lipgloss.NewStyle().
				Foreground(ColorSecondary).
				Bold(true).
				Padding(0, 1)

	TableCellStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(ColorText)
)

func RenderStat(key, value string) string {
	return lipgloss.JoinHorizontal(lipgloss.Left,
		StatKeyStyle.Render(key),
		StatValueStyle.Render(value),
	)
}

func GetTempColor(temp float64) lipgloss.Color {
	if temp < 45 {
		return ColorCyan
	} else if temp < 65 {
		return ColorSuccess
	} else if temp < 85 {
		return ColorWarning
	}
	return ColorError
}

func GetTempIcon(temp float64) string {
	if temp < 45 {
		return "" // Low
	} else if temp < 65 {
		return "" // Normal
	} else if temp < 85 {
		return "" // High
	}
	return "" // Critical
}

func RenderProgressBar(width int, percentage float64) string {
	if percentage < 0 {
		percentage = 0
	}
	if percentage > 100 {
		percentage = 100
	}

	filled := int(float64(width) * percentage / 100)
	empty := width - filled

	bar := ""
	for i := 0; i < filled; i++ {
		// Calculate position ratio relative to total width
		ratio := float64(i) / float64(width-1)
		if width <= 1 {
			ratio = 0
		}

		color := getGradientColor(ratio)
		bar += lipgloss.NewStyle().Foreground(color).Render("█")
	}

	if empty > 0 {
		bar += lipgloss.NewStyle().Foreground(ColorSubtext).Render(repeat("░", empty))
	}

	return "[" + bar + "]"
}

func repeat(s string, count int) string {
	res := ""
	for i := 0; i < count; i++ {
		res += s
	}
	return res
}

// Gradient Helpers
type rgb struct {
	r, g, b int
}

var (
	rgbCyan   = rgb{139, 233, 253} // #8be9fd
	rgbGreen  = rgb{80, 250, 123}  // #50fa7b
	rgbOrange = rgb{255, 184, 108} // #ffb86c
	rgbRed    = rgb{255, 85, 85}   // #ff5555
)

func getGradientColor(t float64) lipgloss.Color {
	// Segments:
	// 0.0 - 0.33: Cyan -> Green
	// 0.33 - 0.66: Green -> Orange
	// 0.66 - 1.0: Orange -> Red

	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}

	var start, end rgb
	var localT float64

	if t < 0.33 {
		start = rgbCyan
		end = rgbGreen
		localT = t / 0.33
	} else if t < 0.66 {
		start = rgbGreen
		end = rgbOrange
		localT = (t - 0.33) / 0.33
	} else {
		start = rgbOrange
		end = rgbRed
		localT = (t - 0.66) / 0.34
	}

	return interpolate(start, end, localT)
}

func interpolate(c1, c2 rgb, t float64) lipgloss.Color {
	r := int(float64(c1.r) + float64(c2.r-c1.r)*t)
	g := int(float64(c1.g) + float64(c2.g-c1.g)*t)
	b := int(float64(c1.b) + float64(c2.b-c1.b)*t)
	// Manual hex formatting since we can't add fmt import easily in same block if strict?
	// Actually I'll do a separate call to add imports.
	// But wait, I can use lipgloss.Color(r,g,b)? No, it takes string.
	// I need fmt.
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
}
