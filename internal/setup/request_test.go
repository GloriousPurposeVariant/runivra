package setup

import "testing"

func TestProblemsAcceptsValidRequest(t *testing.T) {
	req := Request{Environment: "dev", Version: "19.0"}
	problems := req.Problems()
	if len(problems) != 0 {
		t.Fatalf("got problems %v, want none", problems)
	}
}

func TestProblemsReportsMissingOptions(t *testing.T) {
	req := Request{}
	problems := req.Problems()
	if len(problems) != 2 {
		t.Fatalf("got %d problems, want 2: %v", len(problems), problems)
	}
}

func TestProblemsRejectsUnknownEnvironment(t *testing.T) {
	req := Request{Environment: "banana", Version: "19.0"}
	problems := req.Problems()
	if len(problems) != 1 {
		t.Fatalf("got %d problems, want 1: %v", len(problems), problems)
	}
}
