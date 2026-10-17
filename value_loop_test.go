package scalar

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

type valueLoopNode struct {
	Name string
	Next *valueLoopNode
}

// A Go value that contains itself can't become a scalar, as it can't become
// JSON: Value panics, saying which type, where it would have recursed until
// the stack overflowed, which no recover catches. Deep data that doesn't
// loop is fine. In a child process, as a stack overflow kills the process.
func TestValueLoops(t *testing.T) {
	if os.Getenv("SCALAR_VALUE_LOOP") != "" {
		node := &valueLoopNode{Name: "a"}
		node.Next = node
		m := map[string]any{"a": 1}
		m["self"] = m
		s := []any{1, nil}
		s[1] = s
		for name, tc := range map[string]struct {
			v    any
			want string
		}{
			"a struct": {node, "*scalar.valueLoopNode"},
			"a map":    {m, "map[string]interface {}"},
			"a slice":  {s, "[]interface {}"},
		} {
			got := panicOf(func() { Value(tc.v) })
			if !strings.Contains(got, "refers to itself") || !strings.Contains(got, tc.want) {
				t.Errorf("%s: panic %q, want one naming %s", name, got, tc.want)
			}
		}
		var deep any = "leaf"
		for range maxValueDepth - 1 {
			deep = []any{deep}
		}
		if got := panicOf(func() { Value(deep) }); got != "" {
			t.Errorf("%d levels should be fine: %s", maxValueDepth, got)
		}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestValueLoops$", "-test.v")
	cmd.Env = append(os.Environ(), "SCALAR_VALUE_LOOP=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		msg := string(out)
		if len(msg) > 600 {
			msg = msg[:600] + "..."
		}
		t.Fatalf("the child process failed (%v):\n%s", err, msg)
	}
}
