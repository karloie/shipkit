package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

// TaskfileTask represents a task in a Taskfile
type TaskfileTask struct {
	Name         string
	Dependencies []string
	Commands     []string
}

// TaskfileGraph represents the parsed Taskfile dependency graph
type TaskfileGraph struct {
	Tasks map[string]*TaskfileTask
}

// taskfileYAML represents the structure of a Taskfile YAML
type taskfileYAML struct {
	Version string                 `yaml:"version"`
	Tasks   map[string]taskYAMLDef `yaml:"tasks"`
}

type taskYAMLDef struct {
	Deps []string      `yaml:"deps"`
	Cmds []interface{} `yaml:"cmds"` // Can be string or map
}

// ParseTaskfile parses a Taskfile.yml and extracts task dependencies
func ParseTaskfile(path string) (*TaskfileGraph, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var tf taskfileYAML
	if err := yaml.Unmarshal(data, &tf); err != nil {
		return nil, err
	}

	graph := &TaskfileGraph{Tasks: make(map[string]*TaskfileTask)}

	for name, taskDef := range tf.Tasks {
		task := &TaskfileTask{
			Name:         name,
			Dependencies: taskDef.Deps,
			Commands:     []string{},
		}

		// Extract commands (can be strings or maps with 'cmd' key)
		for _, cmd := range taskDef.Cmds {
			switch v := cmd.(type) {
			case string:
				task.Commands = append(task.Commands, v)
			case map[string]interface{}:
				if cmdStr, ok := v["cmd"].(string); ok {
					task.Commands = append(task.Commands, cmdStr)
				}
			}
		}

		graph.Tasks[name] = task
	}

	return graph, nil
}

// GetDependencyTree returns a flattened list of all tasks needed to execute the given task
// in execution order (dependencies before tasks that depend on them)
func (g *TaskfileGraph) GetDependencyTree(task string) []string {
	visited := make(map[string]bool)
	var result []string

	var visit func(string)
	visit = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true

		t, exists := g.Tasks[name]
		if !exists {
			return
		}

		// Visit dependencies first (depth-first)
		for _, dep := range t.Dependencies {
			visit(dep)
		}

		result = append(result, name)
	}

	visit(task)
	return result
}

// HasTask returns true if the Taskfile contains the given task
func (g *TaskfileGraph) HasTask(task string) bool {
	_, exists := g.Tasks[task]
	return exists
}

// GetTasks returns all task names
func (g *TaskfileGraph) GetTasks() []string {
	tasks := make([]string, 0, len(g.Tasks))
	for name := range g.Tasks {
		tasks = append(tasks, name)
	}
	return tasks
}
