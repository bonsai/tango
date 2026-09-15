package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type Repo struct {
	Name   string `json:"name"`
	Size   float64 `json:"size"`
	Stars  float64 `json:"stars"`
	Forks  float64 `json:"forks"`
	Issues float64 `json:"open_issues"`
}

type Dataset struct {
	Name   string `json:"name"`
	Target string `json:"target"`
	Rows   []map[string]any `json:"rows"`
}

type Result struct {
	Question string `json:"question"`
	Engine   string `json:"engine"`
	Dataset  string `json:"dataset"`
	Rows     int `json:"rows"`
	Output   json.RawMessage `json:"output"`
}

func main() {
	input := flag.String("input", "data/repositories.jsonl", "github-observatory repositories.jsonl")
	engine := flag.String("engine", "mean", "bqmlite-go engine")
	target := flag.String("target", "size", "numeric repository field to analyze")
	question := flag.String("question", "What measurable structure exists in the bonsai repository ecosystem?", "research question")
	bqmlite := flag.String("bqmlite", "bqmlite", "path to bqmlite-go CLI")
	out := flag.String("output", "", "optional result JSON")
	flag.Parse()

	ds, err := loadJSONL(*input, *target)
	if err != nil { fatal(err) }

	tmp, err := os.CreateTemp("", "tango-dataset-*.json")
	if err != nil { fatal(err) }
	defer os.Remove(tmp.Name())
	enc := json.NewEncoder(tmp)
	if err := enc.Encode(ds); err != nil { fatal(err) }
	if err := tmp.Close(); err != nil { fatal(err) }

	resultFile := tmp.Name() + ".result.json"
	defer os.Remove(resultFile)
	cmd := exec.Command(*bqmlite, "-input", tmp.Name(), "-engine", *engine, "-output", resultFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fatal(fmt.Errorf("bqmlite-go execution failed: %w", err))
	}
	result, err := os.ReadFile(resultFile)
	if err != nil { fatal(err) }

	wrapped := Result{Question: *question, Engine: *engine, Dataset: ds.Name, Rows: len(ds.Rows), Output: result}
	pretty, err := json.MarshalIndent(wrapped, "", "  ")
	if err != nil { fatal(err) }
	if *out != "" {
		if err := os.WriteFile(*out, append(pretty, '\n'), 0644); err != nil { fatal(err) }
		return
	}
	fmt.Println(string(pretty))
}

func loadJSONL(path, target string) (Dataset, error) {
	f, err := os.Open(path)
	if err != nil { return Dataset{}, err }
	defer f.Close()

	ds := Dataset{Name: "github-observatory/repositories", Target: target}
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" { continue }
		var repo Repo
		if err := json.Unmarshal([]byte(line), &repo); err != nil { return Dataset{}, err }
		ds.Rows = append(ds.Rows, map[string]any{
			"name": repo.Name,
			"size": repo.Size,
			"stars": repo.Stars,
			"forks": repo.Forks,
			"open_issues": repo.Issues,
		})
	}
	if err := s.Err(); err != nil { return Dataset{}, err }
	if len(ds.Rows) == 0 { return Dataset{}, fmt.Errorf("dataset is empty: %s", path) }
	return ds, nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "tango:", err)
	os.Exit(1)
}

var _ = strconv.IntSize
