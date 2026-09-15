package poc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Config struct {
	Research struct {
		Question string `json:"question"`
	} `json:"research"`
}

// ParsePOC intentionally supports the small YAML subset used by poc.yaml.
// This keeps the POC dependency-free; arbitrary YAML is not required yet.
func ParsePOC(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil { return Config{}, err }
	defer f.Close()
	var c Config
	section := ""
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") { continue }
		if line == "research:" { section = "research"; continue }
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") { section = ""; continue }
		if section == "research:" { section = "research" }
		if section == "research" && strings.HasPrefix(line, "question:") {
			c.Research.Question = strings.TrimSpace(strings.TrimPrefix(line, "question:"))
			c.Research.Question = strings.Trim(c.Research.Question, "\"'")
		}
	}
	if err := s.Err(); err != nil { return Config{}, err }
	if c.Research.Question == "" { return Config{}, fmt.Errorf("research.question missing in %s", path) }
	return c, nil
}

type Repo struct {
	Name string `json:"name"`
	Size float64 `json:"size"`
	Stars float64 `json:"stars"`
	Forks float64 `json:"forks"`
	Issues float64 `json:"open_issues"`
}

type Dataset struct {
	Name string `json:"name"`
	Target string `json:"target"`
	Rows []map[string]any `json:"rows"`
}

func LoadJSONL(path string) (Dataset, error) {
	f, err := os.Open(path); if err != nil { return Dataset{}, err }; defer f.Close()
	d := Dataset{Name: "github-observatory/repositories", Target: "size"}
	s := bufio.NewScanner(f); s.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for s.Scan() {
		line := strings.TrimSpace(s.Text()); if line == "" { continue }
		var r Repo
		if err := json.Unmarshal([]byte(line), &r); err != nil { return Dataset{}, err }
		d.Rows = append(d.Rows, map[string]any{"name":r.Name,"size":r.Size,"stars":r.Stars,"forks":r.Forks,"open_issues":r.Issues})
	}
	if err := s.Err(); err != nil { return Dataset{}, err }
	if len(d.Rows) == 0 { return Dataset{}, fmt.Errorf("dataset is empty") }
	return d, nil
}

func Run(question, bqmlite, input string) ([]byte, error) {
	ds, err := LoadJSONL(input); if err != nil { return nil, err }
	tmp, err := os.CreateTemp("", "tango-dataset-*.json"); if err != nil { return nil, err }
	defer os.Remove(tmp.Name())
	if err := json.NewEncoder(tmp).Encode(ds); err != nil { return nil, err }
	if err := tmp.Close(); err != nil { return nil, err }
	resultFile := tmp.Name()+".result.json"; defer os.Remove(resultFile)
	cmd := exec.Command(bqmlite, "-input", tmp.Name(), "-engine", "mean", "-output", resultFile)
	if out, err := cmd.CombinedOutput(); err != nil { return nil, fmt.Errorf("bqmlite-go: %w: %s", err, out) }
	output, err := os.ReadFile(resultFile); if err != nil { return nil, err }
	var raw json.RawMessage = output
	result := struct { Question string `json:"question"`; Engine string `json:"engine"`; Dataset string `json:"dataset"`; Rows int `json:"rows"`; Output json.RawMessage `json:"output"` }{question,"mean",ds.Name,len(ds.Rows),raw}
	return json.MarshalIndent(result,"","  ")
}
