---
spec: pose-done-spec-keeps-its-sealed-plan
category: fixed
breaking: false
---
The approval of a spec that is the only member of a milestone with an exit gate stays valid when the spec turns done: the gate criterion remains in the plan of a done spec whose governing sealed bundle carries it, instead of dropping out and superseding the bundle. `pose close` also stops writing a checkpoint when the terminal verification of a done scope stops on an unknown governance read view, so the next reviewer is not shown an open defect that the next plan recomputes.
