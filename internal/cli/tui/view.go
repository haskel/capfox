package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the TUI
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var sections []string

	// Title bar
	sections = append(sections, m.renderTitleBar())

	// Error display
	if m.err != nil {
		sections = append(sections, errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	}

	// Main content
	if m.status != nil {
		// CPU & Memory bar
		sections = append(sections, m.renderCPUMemory())

		// GPU section
		if len(m.status.GPUs) > 0 {
			sections = append(sections, m.renderGPUs())
		}

		// Storage section
		if len(m.status.Storage) > 0 {
			sections = append(sections, m.renderStorage())
		}
	}

	// Task statistics
	if m.stats != nil && len(m.stats.Tasks) > 0 {
		sections = append(sections, m.renderTaskStats())
	}

	// Footer
	sections = append(sections, m.renderFooter())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) renderTitleBar() string {
	title := titleStyle.Render("CAPFOX DASHBOARD")

	refreshInfo := fmt.Sprintf("↻ %s", m.config.RefreshInterval)
	if m.loading {
		refreshInfo = "↻ loading..."
	}

	help := helpStyle.Render("q:quit r:refresh ↑↓:scroll")

	// Calculate spacing
	rightPart := fmt.Sprintf("%s | %s", refreshInfo, help)
	spacing := m.width - lipgloss.Width(title) - lipgloss.Width(rightPart) - 2
	if spacing < 1 {
		spacing = 1
	}

	return fmt.Sprintf("%s%s%s", title, strings.Repeat(" ", spacing), helpStyle.Render(rightPart))
}

func (m Model) renderCPUMemory() string {
	cpuBar := m.renderProgressBar("CPU", m.status.CPU.UsagePercent, 20)
	memBar := m.renderProgressBar("Memory", m.status.Memory.UsagePercent, 20)

	return fmt.Sprintf("  %s    %s", cpuBar, memBar)
}

func (m Model) renderProgressBar(label string, percent float64, width int) string {
	filled := int(percent / 100 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}

	color := getProgressColor(percent)
	filledBar := lipgloss.NewStyle().Foreground(color).Render(strings.Repeat("█", filled))
	emptyBar := progressBarEmptyStyle.Render(strings.Repeat("░", width-filled))

	return fmt.Sprintf("%s [%s%s] %5.1f%%", labelStyle.Render(label), filledBar, emptyBar, percent)
}

func (m Model) renderGPUs() string {
	var lines []string
	for _, gpu := range m.status.GPUs {
		name := gpu.Name
		if len(name) > 30 {
			name = name[:30]
		}

		header := sectionHeaderStyle.Render(fmt.Sprintf("  GPU %d: %s", gpu.Index, name))
		lines = append(lines, header)

		usageBar := m.renderProgressBar("Usage", gpu.UsagePercent, 12)

		vramUsedGB := float64(gpu.VRAMUsedBytes) / 1024 / 1024 / 1024
		vramTotalGB := float64(gpu.VRAMTotalBytes) / 1024 / 1024 / 1024
		vramPercent := 0.0
		if vramTotalGB > 0 {
			vramPercent = (vramUsedGB / vramTotalGB) * 100
		}
		vramBar := m.renderProgressBar("VRAM", vramPercent, 12)
		vramInfo := fmt.Sprintf("%.1f/%.1f GB", vramUsedGB, vramTotalGB)

		lines = append(lines, fmt.Sprintf("  %s    %s %s", usageBar, vramBar, valueStyle.Render(vramInfo)))
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderStorage() string {
	var lines []string
	lines = append(lines, sectionHeaderStyle.Render("  Storage"))

	// Sort paths for consistent display
	paths := make([]string, 0, len(m.status.Storage))
	for path := range m.status.Storage {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		disk := m.status.Storage[path]
		usedGB := float64(disk.UsedBytes) / 1024 / 1024 / 1024
		totalGB := float64(disk.TotalBytes) / 1024 / 1024 / 1024

		pathDisplay := path
		if len(pathDisplay) > 6 {
			pathDisplay = pathDisplay[:6]
		}
		pathDisplay = fmt.Sprintf("%-6s", pathDisplay)

		bar := m.renderProgressBar(pathDisplay, disk.UsedPct, 20)
		info := fmt.Sprintf("(%.1f / %.1f GB)", usedGB, totalGB)

		lines = append(lines, fmt.Sprintf("  %s  %s", bar, valueStyle.Render(info)))
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderTaskStats() string {
	var lines []string
	lines = append(lines, sectionHeaderStyle.Render("  Task Statistics"))

	// Header
	header := fmt.Sprintf("  %-20s │ %6s │ %7s │ %7s │ %7s",
		"Task", "Count", "CPU Δ", "Mem Δ", "GPU Δ")
	lines = append(lines, tableHeaderStyle.Render(header))

	// Sort tasks by count
	tasks := make([]*TaskStats, 0, len(m.stats.Tasks))
	for _, t := range m.stats.Tasks {
		tasks = append(tasks, t)
	}
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Count > tasks[j].Count
	})

	// Calculate visible rows based on table offset
	maxVisible := 5 // Show max 5 rows
	start := m.tableOffset
	end := start + maxVisible
	if end > len(tasks) {
		end = len(tasks)
	}
	if start >= len(tasks) {
		start = 0
		end = maxVisible
		if end > len(tasks) {
			end = len(tasks)
		}
	}

	for _, t := range tasks[start:end] {
		taskName := t.Task
		if len(taskName) > 20 {
			taskName = taskName[:17] + "..."
		}

		cpuDelta := formatDelta(t.AvgCPUDelta)
		memDelta := formatDelta(t.AvgMemDelta)
		gpuDelta := formatDelta(t.AvgGPUDelta)

		row := fmt.Sprintf("  %-20s │ %6d │ %7s │ %7s │ %7s",
			taskName, t.Count, cpuDelta, memDelta, gpuDelta)
		lines = append(lines, tableCellStyle.Render(row))
	}

	if len(tasks) > maxVisible {
		scrollInfo := fmt.Sprintf("  [%d-%d of %d tasks]", start+1, end, len(tasks))
		lines = append(lines, helpStyle.Render(scrollInfo))
	}

	return strings.Join(lines, "\n")
}

func formatDelta(delta float64) string {
	if delta == 0 {
		return "-"
	}
	sign := "+"
	style := positiveDeltaStyle
	if delta < 0 {
		sign = ""
		style = negativeDeltaStyle
	}
	return style.Render(fmt.Sprintf("%s%.1f%%", sign, delta))
}

func (m Model) renderFooter() string {
	if m.status == nil {
		return ""
	}

	processes := m.status.Process.TotalProcesses
	threads := m.status.Process.TotalThreads
	updated := m.lastUpdated.Format("15:04:05")

	return helpStyle.Render(fmt.Sprintf(
		"  Processes: %d │ Threads: %s │ Updated: %s",
		processes,
		formatNumber(threads),
		updated,
	))
}

func formatNumber(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%d,%03d", n/1000, n%1000)
	}
	return fmt.Sprintf("%d", n)
}
