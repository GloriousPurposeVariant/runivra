package cli

import (
	"errors"
	"testing"

	"github.com/GloriousPurposeVariant/runivra/internal/setup"
)

func TestResultMark(t *testing.T) {
	if resultMark(nil) != markDone {
		t.Fatal("a step without an error must be marked done")
	}
	if resultMark(setup.ErrNotBuiltYet) != markLater {
		t.Fatal("a step that is not built yet must be marked for later")
	}
	if resultMark(errors.New("disk full")) != markFail {
		t.Fatal("any other error must be marked failed")
	}
}
