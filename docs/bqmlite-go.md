# tango × BQMLite-Go

## Role

`tango` is the orchestration / agent layer. `bqmlite-go` is the deterministic ML execution engine.

```text
AW
 ↓
tango
 ├─ research question
 ├─ dataset selection
 ├─ model selection
 ├─ execution plan
 └─ evaluation / next action
      ↓
 bqmlite-go
 ├─ Dataset
 ├─ Train
 ├─ Model
 ├─ Predict
 └─ PredictionResult
      ↓
 tango
 └─ interpret → next question
```

## Boundary

Tango decides **what to run and why**. BQMLite-Go decides **how to execute the supported ML plan reproducibly**.

The interface should remain machine-readable JSON so AW can hand a task to tango and tango can hand an execution request to BQMLite-Go.

## First use case: github-observatory

```text
GitHub
  ↓
github-observatory
  ↓
dataset.json
  ↓
tango
  ↓
bqmlite-go
  ↓
model / prediction / fingerprint
  ↓
tango
  ↓
portfolio breadth / trend analysis
```

## Execution contract

Conceptual request:

```json
{
  "dataset": "github-observatory",
  "target": "cluster",
  "engine": "...",
  "features": [],
  "purpose": "analyze repository breadth"
}
```

Conceptual result:

```json
{
  "status": "ok",
  "model": {},
  "prediction": {},
  "fingerprint": "sha256:..."
}
```

Do not put ML logic into AW or tango. Keep orchestration in tango and execution in BQMLite-Go.
