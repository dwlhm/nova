package scheduler

func Drain(runtime Runtime) (Runtime, []StepResult) {
	results := make([]StepResult, 0)
	next := runtime
	for {
		advanced, result, ok := Step(next)
		if !ok {
			return next, results
		}
		next = advanced
		results = append(results, result)
	}
}
