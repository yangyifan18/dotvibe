package recipe

type LintResult struct {
	Findings []LintFinding `json:"findings"`
}

func LintArchive(path string, opts LintOptions) (LintResult, error) {
	return LintResult{}, nil
}
