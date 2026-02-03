package dag

import (
	"fmt"
	"sort"

	"github.com/joelfokou/workflow/internal/logger"
	"go.uber.org/zap"
)

// TopologicalSort returns tasks in correct execution order.
func (d *DAG) TopologicalSort() ([]*Task, error) {
	neighborList := make(map[string][]string)
	inDegree := make(map[string]int)
	for name := range d.Tasks {
		inDegree[name] = 0
	}

	for _, t := range d.Tasks {
		for _, dep := range t.DependsOn {
			neighborList[dep] = append(neighborList[dep], t.Name)
			inDegree[t.Name]++
		}
	}

	for dep := range neighborList {
		sort.Strings(neighborList[dep])
	}

	// Initialise queue with tasks having in-degree of 0
	queue := []string{}
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}

	result := []*Task{}

	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		result = append(result, d.Tasks[n])

		for _, neighbor := range neighborList[n] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// If result doesn't include all tasks then a cycle exists
	if len(result) != len(d.Tasks) {
		logger.L().Error("cycle detected in DAG", zap.String("workflow", d.Name))
		return nil, fmt.Errorf("cycle detected in DAG")
	}

	return result, nil
}
