package setup

import (
	"errors"
	"os"
)

var ErrNotBuiltYet = errors.New("this kind of step is not built yet")

func Apply(step Step) error {
	switch step.Kind {
	case KindFolder:
		return os.MkdirAll(step.Target, 0o755)
	default:
		return ErrNotBuiltYet
	}
}
