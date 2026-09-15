package pipeline

import "testing"

func TestRun(t *testing.T) {
	steps := []Step{{ID: "observe", Actor: "github-observatory", Action: "load"}, {ID: "plan", Actor: "tango", Action: "select-model"}, {ID: "evaluate", Actor: "tango", Action: "evaluate"}}
	states, err := Run(steps)
	if err != nil { t.Fatal(err) }
	if len(states) != len(steps) { t.Fatalf("got %d states, want %d", len(states), len(steps)) }
	for _, state := range states { if state.Status != "completed" { t.Fatalf("step %s status=%s", state.ID, state.Status) } }
}

func TestDuplicateStep(t *testing.T) {
	_, err := Run([]Step{{ID: "observe"}, {ID: "observe"}})
	if err == nil { t.Fatal("expected duplicate step error") }
}
