package flows

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
)

// The registry is the server-owned contract for both validation and execution.
type Port struct {
	Name string `json:"name"`
	Type string `json:"type"`
}
type NodeKind struct {
	Type       string   `json:"type"`
	Inputs     []Port   `json:"inputs"`
	Outputs    []Port   `json:"outputs"`
	Config     []string `json:"config"`
	InputMode  string   `json:"input_mode,omitempty"`
	Deprecated bool     `json:"deprecated,omitempty"`
	execute    executor `json:"-"`
}
type Node struct {
	ID     string                     `json:"id"`
	Type   string                     `json:"type"`
	Config map[string]json.RawMessage `json:"config"`
	X      float64                    `json:"x"`
	Y      float64                    `json:"y"`
}
type Edge struct {
	ID         string `json:"id"`
	Source     string `json:"source"`
	SourcePort string `json:"source_port"`
	Target     string `json:"target"`
	TargetPort string `json:"target_port"`
}
type Definition struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}
type Flow struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Revision   int        `json:"revision"`
	Definition Definition `json:"definition"`
	CreatedAt  int64      `json:"created_at"`
	UpdatedAt  int64      `json:"updated_at"`
}

func Kinds() []NodeKind {
	out := make([]NodeKind, 0, len(registry))
	for _, kind := range registry {
		if kind.Inputs == nil {
			kind.Inputs = []Port{}
		}
		if kind.Outputs == nil {
			kind.Outputs = []Port{}
		}
		if kind.Config == nil {
			kind.Config = []string{}
		}
		out = append(out, kind)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Type < out[j].Type })
	return out
}

func Validate(name string, definition Definition) error {
	if len(strings.TrimSpace(name)) < 1 || len(name) > 100 {
		return errors.New("flow name must be 1–100 characters")
	}
	if len(definition.Nodes) < 1 || len(definition.Nodes) > 40 || len(definition.Edges) > 80 {
		return errors.New("flow must have 1–40 nodes and at most 80 connections")
	}
	nodes := make(map[string]Node, len(definition.Nodes))
	triggers := 0
	for _, node := range definition.Nodes {
		kind, ok := registry[node.Type]
		if !ok {
			return fmt.Errorf("unknown node type %q", node.Type)
		}
		if !validID(node.ID) || math.IsNaN(node.X) || math.IsNaN(node.Y) || math.IsInf(node.X, 0) || math.IsInf(node.Y, 0) || math.Abs(node.X) > 100000 || math.Abs(node.Y) > 100000 {
			return fmt.Errorf("invalid node identity or position")
		}
		if _, exists := nodes[node.ID]; exists {
			return fmt.Errorf("duplicate node %q", node.ID)
		}
		if strings.HasPrefix(node.Type, "trigger.") {
			triggers++
		}
		if err := validateConfig(kind, node.Config); err != nil {
			return fmt.Errorf("node %s: %w", node.ID, err)
		}
		nodes[node.ID] = node
	}
	if triggers != 1 {
		return errors.New("flow requires exactly one trigger")
	}
	edges := make(map[string]bool, len(definition.Edges))
	incoming := make(map[string]bool)
	incomingType := make(map[string]string)
	incomingCount := make(map[string]int)
	adjacency := make(map[string][]string)
	for _, edge := range definition.Edges {
		if !validID(edge.ID) || edges[edge.ID] {
			return errors.New("invalid or duplicate connection ID")
		}
		edges[edge.ID] = true
		source, a := nodes[edge.Source]
		target, b := nodes[edge.Target]
		if !a || !b || edge.Source == edge.Target {
			return fmt.Errorf("connection %s has invalid endpoints", edge.ID)
		}
		out, okOut := findPort(registry[source.Type].Outputs, edge.SourcePort)
		in, okIn := findPort(registry[target.Type].Inputs, edge.TargetPort)
		if !okOut || !okIn || out.Type != in.Type {
			return fmt.Errorf("connection %s has incompatible ports", edge.ID)
		}
		key := edge.Target + "/" + edge.TargetPort
		if incoming[key] {
			return fmt.Errorf("node %s input %s has multiple connections", edge.Target, edge.TargetPort)
		}
		incoming[key] = true
		incomingType[edge.Target] = in.Type
		incomingCount[edge.Target]++
		adjacency[edge.Source] = append(adjacency[edge.Source], edge.Target)
	}
	for _, node := range definition.Nodes {
		kind := registry[node.Type]
		if kind.InputMode == "one" {
			if incomingCount[node.ID] != 1 {
				return fmt.Errorf("node %s requires exactly one input", node.ID)
			}
			if node.Type == "text.replace" {
				attribute := config(node, "attribute")
				inputType := incomingType[node.ID]
				if attribute != "" && !replaceAttributeAllowed(inputType, attribute) {
					return fmt.Errorf("node %s attribute is incompatible with its input", node.ID)
				}
				for _, edge := range definition.Edges {
					if edge.Source == node.ID {
						out, _ := findPort(kind.Outputs, edge.SourcePort)
						if out.Type != inputType {
							return fmt.Errorf("node %s output is incompatible with its input", node.ID)
						}
					}
				}
			}
			continue
		}
		for _, port := range registry[node.Type].Inputs {
			if !incoming[node.ID+"/"+port.Name] {
				return fmt.Errorf("node %s has unconnected input %s", node.ID, port.Name)
			}
		}
	}
	state := map[string]int{}
	var visit func(string) error
	visit = func(id string) error {
		if state[id] == 1 {
			return errors.New("flow contains a cycle")
		}
		if state[id] == 2 {
			return nil
		}
		state[id] = 1
		for _, next := range adjacency[id] {
			if err := visit(next); err != nil {
				return err
			}
		}
		state[id] = 2
		return nil
	}
	for id := range nodes {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}

func replaceAttributeAllowed(inputType, attribute string) bool {
	switch inputType {
	case "event":
		return attribute == "show_name"
	case "candidate":
		return attribute == "name" || attribute == "provider"
	case "text":
		return attribute == "" || attribute == "value"
	}
	return false
}

func validID(id string) bool {
	if len(id) < 1 || len(id) > 64 {
		return false
	}
	for _, c := range id {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}
func findPort(ports []Port, name string) (Port, bool) {
	for _, p := range ports {
		if p.Name == name {
			return p, true
		}
	}
	return Port{}, false
}
func validateConfig(kind NodeKind, config map[string]json.RawMessage) error {
	if kind.Type == "text.replace" && len(config["find"]) == 0 {
		return errors.New("find text is required")
	}
	for key, raw := range config {
		allowed := false
		for _, candidate := range kind.Config {
			if key == candidate {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("unknown configuration %q", key)
		}
		if len(raw) > 1024 {
			return fmt.Errorf("configuration %q is too large", key)
		}
		var value string
		if err := json.Unmarshal(raw, &value); err != nil || len(value) > 500 {
			return fmt.Errorf("configuration %q must be short text", key)
		}
		if kind.Type == "text.replace" && key == "find" && value == "" {
			return errors.New("find text is required")
		}
		if kind.Type == "text.replace" && key == "attribute" && value != "" && value != "value" && value != "show_name" && value != "name" && value != "provider" {
			return errors.New("unknown replacement attribute")
		}
		if kind.Type == "text.replace" && key == "trim" && value != "true" && value != "false" {
			return errors.New("trim must be true or false")
		}
		if kind.Type == "torrent.filter" && key == "min_seeders" {
			if value != "" {
				var n int
				if _, err := fmt.Sscanf(value, "%d", &n); err != nil || n < 0 || n > 10000 || fmt.Sprint(n) != value {
					return errors.New("min seeders must be 0–10000")
				}
			}
		}
	}
	return nil
}
