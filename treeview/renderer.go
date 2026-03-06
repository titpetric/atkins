package treeview

import (
	"fmt"
	"strings"
	"sync"

	"github.com/titpetric/atkins/colors"
)

// DefaultMaxArgLen is the default maximum length for argument values before compaction.
const DefaultMaxArgLen = 25

// Renderer handles rendering of tree nodes to strings with proper formatting.
type Renderer struct {
	mu        sync.Mutex
	trimmer   *Trimmer
	maxArgLen int
}

// NewRenderer creates a new tree renderer.
func NewRenderer() *Renderer {
	return &Renderer{
		trimmer:   NewTrimmer(),
		maxArgLen: DefaultMaxArgLen,
	}
}

// trimLabel applies argument compaction and viewport trimming to a label.
func (r *Renderer) trimLabel(label string, prefixLen int) string {
	if r.trimmer == nil {
		return label
	}
	return r.trimmer.TrimLabel(label, r.maxArgLen, prefixLen)
}

// Render converts a node to a string representation during execution (shows status for all nodes).
func (r *Renderer) Render(root *Node) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Refresh viewport width before each render
	if r.trimmer != nil {
		r.trimmer.RefreshViewport()
	}

	output := colors.BrightWhite(root.GetName()) + "\n"

	if root.IsSummarize() {
		output += r.renderNodeSummary(root, "", true)
		return output
	}

	children := root.GetChildren()
	for i, child := range children {
		isLast := i == len(children)-1
		output += r.renderNodeForExecution(child, "", isLast)
	}

	return output
}

// RenderStatic renders a static tree (for list views) without spinners.
func (r *Renderer) RenderStatic(root *Node) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	output := colors.BrightWhite(root.GetName()) + "\n"

	if root.IsSummarize() {
		output += r.renderNodeSummary(root, "", true)
		return output
	}

	children := root.GetChildren()
	for i, child := range children {
		isLast := i == len(children)-1
		output += r.renderStaticNode(child, "", isLast)
	}

	return output
}

// renderNodeSummary will give a one-liner with status (pending, running, passed...)
func (r *Renderer) renderNodeSummary(node *Node, prefix string, isLast bool) string {
	// Determine branch character
	branch := "├─ "
	if isLast {
		branch = "└─ "
	}

	var pending, running, passing, failed int
	for _, child := range node.GetChildren() {
		switch child.GetStatus() {
		case StatusRunning:
			running++
		case StatusFailed:
			failed++
		case StatusPending:
			pending++
		case StatusPassed:
			passing++
		}
	}

	total := pending + running + passing + failed

	// Trim label to fit viewport (prefix + branch = indentation)
	prefixLen := colors.VisualLength(prefix + branch)

	// If no children, just show the node name
	if total == 0 {
		label := node.Label()
		status := node.StatusColor()
		if status != "" {
			label = label + " " + status
		}
		label = r.trimLabel(label, prefixLen)
		return prefix + branch + label + "\n"
	}

	summary := colors.White(fmt.Sprintf("%d/%d", passing, total))
	if total == passing {
		summary = colors.Green(fmt.Sprintf("%d/%d", passing, total))
	}

	label := node.Label() + " " + node.StatusColor() + " (" + summary + ")"
	label = r.trimLabel(label, prefixLen)
	return prefix + branch + label + "\n"
}

// renderNodeForExecution renders a node during execution, showing status for all nodes including steps.
func (r *Renderer) renderNodeForExecution(node *Node, prefix string, isLast bool) string {
	output := ""

	// Determine branch character
	branch := "├─ "
	if isLast {
		branch = "└─ "
	}

	if node.IsSummarize() {
		return r.renderNodeSummary(node, prefix, isLast)
	}

	if node.IsQuiet() {
		return ""
	}

	label := node.Label()
	status := node.StatusColor()

	// Build the node label with dependencies and deferred info
	deps := node.GetDependencies()
	if len(deps) > 0 {
		depItems := make([]string, len(deps))
		for j, dep := range deps {
			depItems[j] = colors.BrightOrange(dep)
		}
		depsStr := strings.Join(depItems, ", ")
		label = label + fmt.Sprintf(" (depends_on: %s)", depsStr)
	}

	// Add if condition for skipped nodes
	if node.GetStatus() == StatusSkipped {
		if ifCond := node.GetIf(); ifCond != "" {
			label = label + " " + colors.BrightYellow(fmt.Sprintf("(if: %s)", ifCond))
		}
	}

	// Add status indicator - show all status during execution
	if status != "" && !strings.HasSuffix(strings.TrimSpace(label), "●") &&
		!strings.HasSuffix(strings.TrimSpace(label), "✓") &&
		!strings.HasSuffix(strings.TrimSpace(label), "✗") {
		label = label + " " + status
	}

	// Get children once for consistent progress counter and rendering
	children := node.GetChildren()

	// Add progress counter for nodes with multiple children
	if len(children) > 1 {
		var passing, total int
		for _, child := range children {
			total++
			if child.GetStatus() == StatusPassed {
				passing++
			}
		}
		progress := colors.White(fmt.Sprintf("%d/%d", passing, total))
		if total == passing {
			progress = colors.Green(fmt.Sprintf("%d/%d", passing, total))
		}
		label = label + " (" + progress + ")"
	}

	// Trim label to fit viewport (prefix + branch = indentation)
	prefixLen := colors.VisualLength(prefix + branch)
	label = r.trimLabel(label, prefixLen)

	// Render this node
	output += prefix + branch + label
	output += "\n"

	// Render output lines from command execution (with proper indentation)
	// Use GetOutput() for thread-safe access to output slice
	nodeOutput := node.GetOutput()
	if len(nodeOutput) > 0 {
		// Determine continuation character for output indentation
		continuation := "│  "
		if isLast {
			continuation = "   "
		}

		// Trim output lines and calculate max width for border (visual width, excluding ANSI)
		outputPrefixLen := colors.VisualLength(prefix + continuation)
		hasBorder := len(nodeOutput) >= 2
		// Account for border characters: │ content │ adds 4 visual chars (┌/└, space, space, ┐/┘)
		borderOverhead := 0
		if hasBorder {
			borderOverhead = 4
		}
		trimmedLines := make([]string, len(nodeOutput))
		maxWidth := 0
		for i, outputLine := range nodeOutput {
			trimmedLine := r.trimLabel(outputLine, outputPrefixLen+borderOverhead)
			trimmedLines[i] = trimmedLine
			width := colors.VisualLength(trimmedLine)
			if width > maxWidth {
				maxWidth = width
			}
		}

		// Add top border if 2+ elements (account for spaces around content)
		if hasBorder {
			topBorder := prefix + continuation + colors.Gray("┌"+strings.Repeat("─", maxWidth+2)+"┐") + "\n"
			output += topBorder
		}

		// Add each output line with left/right borders
		for _, trimmedLine := range trimmedLines {
			// Pad line to max width for consistent border (using visual width)
			currentWidth := colors.VisualLength(trimmedLine)
			padding := strings.Repeat(" ", maxWidth-currentWidth)
			paddedLine := " " + trimmedLine + padding + " "
			if hasBorder {
				output += prefix + continuation + colors.Gray("│") + colors.White(paddedLine) + colors.Gray("│") + "\n"
			} else {
				output += prefix + continuation + colors.White(trimmedLine) + "\n"
			}
		}

		// Add bottom border if 2+ elements (account for spaces around content)
		if hasBorder {
			bottomBorder := prefix + continuation + colors.Gray("└"+strings.Repeat("─", maxWidth+2)+"┘") + "\n"
			output += bottomBorder
		}
	}

	// Render children
	if len(children) > 0 {
		// Determine continuation character
		continuation := "│  "
		if isLast {
			continuation = "   "
		}

		for j, child := range children {
			childIsLast := j == len(children)-1
			output += r.renderNodeForExecution(child, prefix+continuation, childIsLast)
		}
	}

	return output
}

// renderStaticNode renders a static node without execution state (for list views)
func (r *Renderer) renderStaticNode(node *Node, prefix string, isLast bool) string {
	output := ""

	// Determine branch character
	branch := "├─ "
	if isLast {
		branch = "└─ "
	}

	if node.IsSummarize() {
		return r.renderNodeSummary(node, prefix, isLast)
	}

	if node.IsQuiet() {
		return ""
	}

	label := node.Label()
	status := node.StatusColor()

	// Build the node label with dependencies and deferred info
	deps := node.GetDependencies()
	if len(deps) > 0 {
		depItems := make([]string, len(deps))
		for j, dep := range deps {
			depItems[j] = colors.BrightOrange(dep)
		}
		depsStr := strings.Join(depItems, ", ")
		label = label + fmt.Sprintf(" (depends_on: %s)", depsStr)
	}

	// Add if condition for skipped nodes
	if node.GetStatus() == StatusSkipped {
		if ifCond := node.GetIf(); ifCond != "" {
			label = label + " " + colors.BrightYellow(fmt.Sprintf("(if: %s)", ifCond))
		}
	}

	// Add status indicator only for jobs, not for steps (in list view)
	nodeName := node.GetName()
	isStep := strings.Contains(nodeName, "task:") || strings.Contains(nodeName, "run:") ||
		strings.Contains(nodeName, "cmd:") || strings.Contains(nodeName, "cmds:")
	if status != "" && !isStep {
		label = label + " " + status
	}

	// Trim label to fit viewport (prefix + branch = indentation)
	prefixLen := colors.VisualLength(prefix + branch)
	label = r.trimLabel(label, prefixLen)

	// Render this node
	output += prefix + branch + label
	output += "\n"

	// Render output lines from command execution (with proper indentation)
	// Use GetOutput() for thread-safe access to output slice
	nodeOutput := node.GetOutput()
	if len(nodeOutput) > 0 {
		// Determine continuation character for output indentation
		continuation := "│  "
		if isLast {
			continuation = "   "
		}

		// Trim output lines and calculate max width for border (visual width, excluding ANSI)
		outputPrefixLen := colors.VisualLength(prefix + continuation)
		hasBorder := len(nodeOutput) >= 2
		// Account for border characters: │ content │ adds 4 visual chars (┌/└, space, space, ┐/┘)
		borderOverhead := 0
		if hasBorder {
			borderOverhead = 4
		}
		trimmedLines := make([]string, len(nodeOutput))
		maxWidth := 0
		for i, outputLine := range nodeOutput {
			trimmedLine := r.trimLabel(outputLine, outputPrefixLen+borderOverhead)
			trimmedLines[i] = trimmedLine
			width := colors.VisualLength(trimmedLine)
			if width > maxWidth {
				maxWidth = width
			}
		}

		// Add top border if 2+ elements (account for spaces around content)
		if hasBorder {
			topBorder := prefix + continuation + colors.Gray("┌"+strings.Repeat("─", maxWidth+2)+"┐") + "\n"
			output += topBorder
		}

		// Add each output line with left/right borders
		for _, trimmedLine := range trimmedLines {
			// Pad line to max width for consistent border (using visual width)
			currentWidth := colors.VisualLength(trimmedLine)
			padding := strings.Repeat(" ", maxWidth-currentWidth)
			paddedLine := " " + trimmedLine + padding + " "
			if hasBorder {
				output += prefix + continuation + colors.Gray("│") + colors.White(paddedLine) + colors.Gray("│") + "\n"
			} else {
				output += prefix + continuation + colors.White(trimmedLine) + "\n"
			}
		}

		// Add bottom border if 2+ elements (account for spaces around content)
		if hasBorder {
			bottomBorder := prefix + continuation + colors.Gray("└"+strings.Repeat("─", maxWidth+2)+"┘") + "\n"
			output += bottomBorder
		}
	}

	// Render children
	children := node.GetChildren()
	if len(children) > 0 {
		// Determine continuation character
		continuation := "│  "
		if isLast {
			continuation = "   "
		}

		for j, child := range children {
			childIsLast := j == len(children)-1
			output += r.renderStaticNode(child, prefix+continuation, childIsLast)
		}
	}

	return output
}

// CountLines returns the number of lines the tree will render.
func CountLines(root *Node) int {
	count := 1 // root line
	children := root.GetChildren()
	for _, child := range children {
		count += countNodeLines(child)
	}
	return count
}

func countNodeLines(node *Node) int {
	count := 1 // this node
	children := node.GetChildren()
	for _, child := range children {
		count += countNodeLines(child)
	}
	return count
}
