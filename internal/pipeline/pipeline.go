package pipeline

import "fmt"

type Step struct {
	ID     string
	Actor  string
	Action string
}

type State struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func Validate(steps []Step) error {
	if len(steps) == 0 { return fmt.Errorf("pipeline is empty") }
	seen := map[string]bool{}
	for _, step := range steps {
		if step.ID == "" { return fmt.Errorf("pipeline step id is empty") }
		if seen[step.ID] { return fmt.Errorf("duplicate pipeline step: %s", step.ID) }
		seen[step.ID] = true
	}
	return nil
}

func Run(steps []Step) ([]State, error) {
	if err := Validate(steps); err != nil { return nil, err }
	states := make([]State, 0, len(steps))
	for _, step := range steps {
		states = append(states, State{ID: step.ID, Status: "completed"})
	}
	return states, nil
}
