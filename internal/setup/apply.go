package setup

import (
	"context"
	"errors"
	"os"

	"github.com/GloriousPurposeVariant/runivra/internal/project"
)

var ErrNotBuiltYet = errors.New("this kind of step is not built yet")

func Apply(ctx context.Context, step Step, req Request, report func(Progress)) error {

	switch step.Kind {
	case KindFolder:
		return os.MkdirAll(step.Target, 0o755)
	case KindFile:
		if step.Template == "" {
			return ErrNotBuiltYet
		}
		return writeFile(step, req)
	case KindClone:
		return clone(ctx, step, report)

	case KindCopy:
		return copyFolder(ctx, step.Source, step.Target)

	case KindProject:
		return project.Save(req.Folder(), projectConfig(req))

	default:
		return ErrNotBuiltYet
	}
}
