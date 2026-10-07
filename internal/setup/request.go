package setup

type Request struct {
	Environment string
	Version     string
}

func (r Request) Missing() []string {
	var missing []string
	if r.Environment == "" {
		missing = append(missing, "--env dev|staging|production")
	}
	if r.Version == "" {
		missing = append(missing, "--version 19.0")
	}
	return missing
}
