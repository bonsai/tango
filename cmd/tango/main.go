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

type Repo struct { Name string `json:"name"`; Size float64 `json:"size"`; Stars float64 `json:"stars"`; Forks float64 `json:"forks"`; Issues float64 `json:"open_issues"` }
type Dataset struct { Name string `json:"name"`; Target string `json:"target"`; Rows []map[string]any `json:"rows"` }
type POC struct { Question string }

func main() {
	poc := flag.String("poc", "poc.yaml", "POC YAML plan")
	input := flag.String("input", "data/repositories.jsonl", "github-observatory repositories.jsonl")
	bqmlite := flag.String("bqmlite", "bqmlite", "path to bqmlite-go CLI")
	out := flag.String("output", "", "optional result JSON")
	flag.Parse()

	plan, err := loadPOC(*poc); if err != nil { fatal(err) }
	ds, err := loadJSONL(*input); if err != nil { fatal(err) }
	tmp, err := os.CreateTemp("", "tango-dataset-*.json"); if err != nil { fatal(err) }; defer os.Remove(tmp.Name())
	if err := json.NewEncoder(tmp).Encode(ds); err != nil { fatal(err) }; if err := tmp.Close(); err != nil { fatal(err) }
	resultFile := tmp.Name()+".result.json"; defer os.Remove(resultFile)
	cmd := exec.Command(*bqmlite, "-input", tmp.Name(), "-engine", "mean", "-output", resultFile)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil { fatal(fmt.Errorf("bqmlite-go execution failed: %w", err)) }
	output, err := os.ReadFile(resultFile); if err != nil { fatal(err) }
	result := struct { Question string `json:"question"`; Engine string `json:"engine"`; Dataset string `json:"dataset"`; Rows int `json:"rows"`; Output json.RawMessage `json:"output"` }{plan.Question,"mean",ds.Name,len(ds.Rows),json.RawMessage(output)}
	pretty, err := json.MarshalIndent(result,"","  "); if err != nil { fatal(err) }
	if *out != "" { if err := os.WriteFile(*out, append(pretty,'\n'),0644); err != nil { fatal(err) }; return }; fmt.Println(string(pretty))
}

func loadPOC(path string) (POC, error) {
	f, err := os.Open(path); if err != nil { return POC{}, err }; defer f.Close()
	var p POC; section := ""; s := bufio.NewScanner(f)
	for s.Scan() { raw:=s.Text(); line:=strings.TrimSpace(raw); if line==""||strings.HasPrefix(line,"#"){continue}; indent:=len(raw)-len(strings.TrimLeft(raw," \t")); if indent==0&&strings.HasSuffix(line,":"){section=strings.TrimSuffix(line,":");continue}; if section=="research"&&indent>0&&strings.HasPrefix(line,"question:"){p.Question=strings.Trim(strings.TrimSpace(strings.TrimPrefix(line,"question:")),"\"'")} }
	if err:=s.Err();err!=nil{return POC{},err}; if p.Question==""{return POC{},fmt.Errorf("research.question missing in %s",path)}; return p,nil
}

func loadJSONL(path string) (Dataset,error) {
	f,err:=os.Open(path);if err!=nil{return Dataset{},err};defer f.Close(); d:=Dataset{Name:"github-observatory/repositories",Target:"size"}; s:=bufio.NewScanner(f);s.Buffer(make([]byte,64*1024),4*1024*1024)
	for s.Scan(){line:=strings.TrimSpace(s.Text());if line==""{continue};var r Repo;if err:=json.Unmarshal([]byte(line),&r);err!=nil{return Dataset{},err};d.Rows=append(d.Rows,map[string]any{"name":r.Name,"size":r.Size,"stars":r.Stars,"forks":r.Forks,"open_issues":r.Issues})};if err:=s.Err();err!=nil{return Dataset{},err};if len(d.Rows)==0{return Dataset{},fmt.Errorf("dataset is empty: %s",path)};return d,nil
}
func fatal(err error){fmt.Fprintln(os.Stderr,"tango:",err);os.Exit(1)}
