package storage

import (
	"context"
	"sort"
)

// ToolStat is one tool and what it has cost across every form of it, a
// row of the tool report. `go` is one tool whether it builds, tests or
// vets, and its share of a week is the week's share of Go.
type ToolStat struct {
	// Tool is the first word of the command key.
	Tool string

	// Commands is how many distinct command keys of it ran.
	Commands int64

	// Runs is how many times any of them ran.
	Runs int64

	// TotalMs is the time they have taken altogether.
	TotalMs int64

	// Failures is how many of the runs exited non-zero.
	Failures int64
}

// Tools returns the query error, or the command keys rolled up per tool,
// the most expensive first.
func (s *Storage) Tools(ctx context.Context, req StatsRequest) ([]ToolStat, error) {
	stats, err := s.keyed(ctx, req)
	if err != nil {
		return nil, err
	}

	var (
		order  []string
		byTool = map[string]*ToolStat{}
	)
	for _, stat := range stats {
		tool, ok := byTool[stat.Tool]
		if !ok {
			tool = &ToolStat{Tool: stat.Tool}
			byTool[stat.Tool] = tool
			order = append(order, stat.Tool)
		}
		tool.Commands++
		tool.Runs += stat.Runs
		tool.TotalMs += stat.TotalMs
		tool.Failures += stat.Failures
	}

	tools := make([]ToolStat, 0, len(order))
	for _, name := range order {
		tools = append(tools, *byTool[name])
	}
	sort.SliceStable(tools, func(i, j int) bool {
		if tools[i].TotalMs != tools[j].TotalMs {
			return tools[i].TotalMs > tools[j].TotalMs
		}
		if tools[i].Runs != tools[j].Runs {
			return tools[i].Runs > tools[j].Runs
		}
		return tools[i].Tool < tools[j].Tool
	})

	return tools[:min(len(tools), limitOf(req.Limit))], nil
}
