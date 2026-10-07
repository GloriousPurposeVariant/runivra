package setup

import (
	"errors"
	"os"
)

var ErrNotBuiltYet = errors.New("this kind of step is not built yet")

func Apply(step Step, req Request) error {
	switch step.Kind {
	case KindFolder:
		return os.MkdirAll(step.Target, 0o755)
	case KindFile:
		if step.Template == "" {
			return ErrNotBuiltYet
		}
		return writeFile(step, req)
	default:
		return ErrNotBuiltYet
	}
}
