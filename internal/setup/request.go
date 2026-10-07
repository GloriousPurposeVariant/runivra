package setup

import "fmt"

type Request struct {
	Environment string
	Version     string
}

func (r Request) Problems() []string {
	var problems []string

	switch r.Environment {
	case "":
		problems = append(problems, "--env is required: dev, staging or prod")
	case "dev", "development", "stage", "staging", "prod", "production":
	default:
		problems = append(problems, fmt.Sprintf("--env %q is not valid: use dev, staging or prod", r.Environment))
	}

	if r.Version == "" {
		problems = append(problems, "--version is required, for example 19.0")
	}
	return problems
}
