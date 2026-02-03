// Package dag provides structures and methods to manage Directed Acyclic Graphs (DAGs) for task scheduling.
package dag

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
)

type Task struct {
	Name      string   `json:"name"`
	Cmd       string   `json:"cmd"`
	DependsOn []string `json:"depends_on"`
	Retries   int      `json:"retries"`
}

type DAG struct {
	Name  string           `json:"name"`
	Tasks map[string]*Task `json:"tasks"`
}

// ComputeHash generates a SHA-256 hash representing the current state of the DAG.
func (d *DAG) ComputeHash() (string, error) {
	type taskSnapshot struct {
		Name      string   `json:"name"`
		Cmd       string   `json:"cmd"`
		DependsOn []string `json:"depends_on"`
		Retries   int      `json:"retries"`
	}

	type dagSnapshot struct {
		Name  string         `json:"name"`
		Tasks []taskSnapshot `json:"tasks"`
	}

	// Create sorted task list for consistent hashing
	var tasks []taskSnapshot
	for _, t := range d.Tasks {
		deps := make([]string, len(t.DependsOn))
		copy(deps, t.DependsOn)
		sort.Strings(deps)
		tasks = append(tasks, taskSnapshot{
			Name:      t.Name,
			Cmd:       t.Cmd,
			DependsOn: deps,
			Retries:   t.Retries,
		})
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Name < tasks[j].Name
	})

	snapshot := dagSnapshot{
		Name:  d.Name,
		Tasks: tasks,
	}

	data, err := json.Marshal(snapshot)
	if err != nil {
		return "", err
	}

	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}

// Graph generates a simple textual representation of the DAG structure.
func (d *DAG) Graph() string {
	out := ""
	for _, t := range d.Tasks {
		if len(t.DependsOn) == 0 {
			out += t.Name + "\n"
			continue
		}
		for _, dep := range t.DependsOn {
			out += dep + " -> " + t.Name + "\n"
		}
	}
	return out
}

// Roots returns all tasks with no dependencies.
func (d *DAG) Roots() []*Task {
	var roots []*Task
	for _, t := range d.Tasks {
		if len(t.DependsOn) == 0 {
			roots = append(roots, t)
		}
	}
	return roots
}

// RenderASCII generates an ASCII representation of the DAG structure.
func (d *DAG) RenderASCII() string {
	var b strings.Builder

	children := map[string][]string{}
	for _, n := range d.Tasks {
		for _, dep := range n.DependsOn {
			children[dep] = append(children[dep], n.Name)
		}
	}

	var render func(node string, prefix string)
	render = func(node string, prefix string) {
		b.WriteString(prefix + node + "\n")
		kids := children[node]
		sort.Strings(kids)
		for i, k := range kids {
			last := i == len(kids)-1
			edge := "├── "
			next := prefix + "│   "
			if last {
				edge = "└── "
				next = prefix + "    "
			}
			b.WriteString(prefix + edge)
			render(k, next)
		}
	}

	roots := d.Roots()
	sort.Slice(roots, func(i, j int) bool {
		return roots[i].Name < roots[j].Name
	})

	for _, root := range roots {
		render(root.Name, "")
	}

	return b.String()
}
