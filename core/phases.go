package core

// GenerationPhase names non-overlapping implemented core boundaries. Reference
// resolution and initial naming occur inside Analyze and are not separate phases.
type GenerationPhase string

const (
	PhaseOptions   GenerationPhase = "options"
	PhaseParse     GenerationPhase = "parse"
	PhaseAnalyze   GenerationPhase = "analyze"
	PhaseTransform GenerationPhase = "transform"
	PhaseEmit      GenerationPhase = "emit"
)

// GenerationPhaseObserver is called synchronously at each entered phase start.
// Its returned function, if nonnil, is called once when that phase ends, including
// an ordinary generation error. Later unentered phases are not reported. A nil
// observer disables observation. It receives no schemas, plans or source bytes.
//
// The caller owns any clock, IO or collected metadata. Callbacks must not panic
// or mutate the request while generation is running; their cost is included in
// the surrounding command duration. Separate concurrent generations need their
// own collector or caller-provided synchronization. Observation is never retained
// in GenerationOptions, reports, warnings, files or name-lock metadata.
type GenerationPhaseObserver func(GenerationPhase) func()

type phaseObserver struct {
	start  GenerationPhaseObserver
	finish func()
}

func (p *phaseObserver) begin(phase GenerationPhase) {
	p.end()
	if p.start != nil {
		p.finish = p.start(phase)
	}
}
func (p *phaseObserver) end() {
	if p.finish != nil {
		finish := p.finish
		p.finish = nil
		finish()
	}
}
