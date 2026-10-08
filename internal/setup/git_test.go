package setup

import "testing"

func TestProgressWriterReadsGitOutput(t *testing.T) {
	var seen []Progress
	writer := &progressWriter{report: func(p Progress) { seen = append(seen, p) }}

	writer.Write([]byte("Cloning into 'odoo'...\nremote: Counting objects:  50% (5/10)\rRecei"))
	writer.Write([]byte("ving objects:  45% (1234/2740), 10.5 MiB | 2.1 MiB/s\rfatal: early EOF\n"))

	if len(seen) != 2 {
		t.Fatalf("got %d progress updates, want 2: %v", len(seen), seen)
	}
	if seen[0].Label != "Counting objects" || seen[0].Percent != 50 {
		t.Fatalf("first update = %+v", seen[0])
	}
	if seen[1].Label != "Receiving objects" || seen[1].Percent != 45 {
		t.Fatalf("second update = %+v", seen[1])
	}
	if writer.lastMessage() != "fatal: early EOF" {
		t.Fatalf("lastMessage() = %q, want the fatal line", writer.lastMessage())
	}
}
