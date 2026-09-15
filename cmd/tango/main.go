package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Repo struct {
	Name string  `json:"name"`
	Size float64 `json:"size"`
	Stars float64 `json:"stars"`
	Forks float64 `json:"forks"`
	Issues float64 `json:"open_issues"`
}

type Dataset struct {
	Name   string           `json:"name"`
	Target string           `json:"target"`
	Rows   []map[string]any `json:"rows"`
}

type Step struct {
	ID     string
	Actor  string
	Action string
}

type POC struct {
	Question string
	Engine   string
	Target   string
	Steps    []Step
}

type State struct {
	ID     string `json:"id"`
	Actor  string `json:"actor"`
	Action string `json:"action"`
	Status string `json:"status"`
}

func main() {
	poc := flag.String("poc", "poc.yaml", "POC YAML plan")
	input := flag.String("input", "data/repositories.jsonl", "github-observatory repositories.jsonl")
	bqmlite := flag.String("bqmlite", "bqmlite", "path to bqmlite-go CLI")
	out := flag.String("output", "", "optional result JSON")
	flag.Parse()

	plan, err := loadPOC(*poc)
	if err != nil { fatal(err) }
	ds, err := loadJSONL(*input)
	if err != nil { fatal(err) }
	if plan.Target != "" { ds.Target = plan.Target }

	states := make([]State, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		states = append(states, State{ID: step.ID, Actor: step.Actor, Action: step.Action, Status: "planned"})
	}
	mark := func(id, status string) {
		for i := range states { if states[i].ID == id { states[i].Status = status } }
	}

	mark("observe", "completed")
	mark("plan", "completed")

	tmp, err := os.CreateTemp("", "tango-dataset-*.json")
	if err != nil { fatal(err) }
	defer os.Remove(tmp.Name())
	if err := json.NewEncoder(tmp).Encode(ds); err != nil { fatal(err) }
	if err := tmp.Close(); err != nil { fatal(err) }

	resultFile := tmp.Name() + ".result.json"
	defer os.Remove(resultFile)
	engine := plan.Engine
	if engine == "" { engine = "mean" }
	cmd := exec.Command(*bqmlite, "-input", tmp.Name(), "-engine", engine, "-output", resultFile)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil { fatal(fmt.Errorf("bqmlite-go execution failed: %w", err)) }
	mark("train", "completed")
	mark("predict", "completed")

	output, err := os.ReadFile(resultFile)
	if err != nil { fatal(err) }
	var mlResult any
	if err := json.Unmarshal(output, &mlResult); err != nil { fatal(err) }

	mark("evaluate", "completed")
	nextQuestion := nextQuestion(plan.Question, engine, len(ds.Rows))
	mark("next", "completed")

	result := struct {
		Question     string `json:"question"`
		Engine       string `json:"engine"`
		Dataset      string `json:"dataset"`
		Target       string `json:"target"`
		Rows         int    `json:"rows"`
		States       []State `json:"states"`
		MLResult     any    `json:"ml_result"`
		NextQuestion string `json:"next_question"`
	}{plan.Question, engine, ds.Name, ds.Target, len(ds.Rows), states, mlResult, nextQuestion}

	pretty, err := json.MarshalIndent(result, "", "  ")
	if err != nil { fatal(err) }
	if *out != "" {
		if err := os.WriteFile(*out, append(pretty, '\n'), 0644); err != nil { fatal(err) }
		return
	}
	fmt.Println(string(pretty))
}

func loadPOC(path string) (POC, error) {
	f, err := os.Open(path)
	if err != nil { return POC{}, err }
	defer f.Close()

	var p POC
	section := ""
	currentStep := -1
	s := bufio.NewScanner(f)
	for s.Scan() {
		raw := s.Text()
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") { continue }
		indent := len(raw) - len(strings.TrimLeft(raw, " \t"))

		if indent == 0 && strings.HasSuffix(line, ":") {
			section = strings.TrimSuffix(line, ":")
			currentStep = -1
			continue
		}
		if indent == 0 { continue }

		if section == "research" && strings.HasPrefix(line, "question:") {
			p.Question = value(line, "question:")
		}
		if section == "data" && strings.HasPrefix(line, "target:") {
			p.Target = value(line, "target:")
		}
		if section == "model" && strings.HasPrefix(line, "engine:") {
			p.Engine = value(line, "engine:")
		}
		if section == "pipeline" && strings.HasPrefix(line, "- id:") {
			p.Steps = append(p.Steps, Step{ID: value(line, "- id:")})
			currentStep = len(p.Steps) - 1
			continue
		}
		if section == "pipeline" && currentStep >= 0 {
			if strings.HasPrefix(line, "actor:") { p.Steps[currentStep].Actor = value(line, "actor:") }
			if strings.HasPrefix(line, "action:") { p.Steps[currentStep].Action = value(line, "action:") }
		}
	}
	if err := s.Err(); err != nil { return POC{}, err }
	if p.Question == "" { return POC{}, fmt.Errorf("research.question missing in %s", path) }
	if len(p.Steps) == 0 { return POC{}, fmt.Errorf("pipeline missing in %s", path) }
	return p, nil
}

func value(line, prefix string) string {
	return strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, prefix)), "\"'")
}

func nextQuestion(question, engine string, rows int) string {
	return fmt.Sprintf("How does %s modeling of %d repository observations refine the question: %s", engine, rows, question)
}

func loadJSONL(path string) (Dataset, error) {
	f, err := os.Open(path)
	if err != nil { return Dataset{}, err }
	defer f.Close()
	d := Dataset{Name: "github-observatory/repositories", Target: "size"}
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" { continue }
		var r Repo
		if err := json.Unmarshal([]byte(line), &r); err != nil { return Dataset{}, err }
		d.Rows = append(d.Rows, map[string]any{"name": r.Name, "size": r.Size, "stars": r.Stars, "forks": r.Forks, "open_issues": r.Issues})
	}
	if err := s.Err(); err != nil { return Dataset{}, err }
	if len(d.Rows) == 0 { return Dataset{}, fmt.Errorf("dataset is empty: %s", path) }
	return d, nil
}

func fatal(err error) { fmt.Fprintln(os.Stderr, "tango:", err); os.Exit(1) }
