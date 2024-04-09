package http

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func Test_AbsURLPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		path     string
		abs      string
		expected error
	}{
		{
			name: "relative to abs",
			path: "my/path",
			abs:  "/my/path",
		},
		{
			name: "abs to abs",
			path: "/my/path",
			abs:  "/my/path",
		},
		{
			name: "relative traversal to abs",
			path: "my/../path",
			abs:  "/path",
		},
		{
			name: "abs traversal to abs",
			path: "/my/../path",
			abs:  "/path",
		},
	}
	for _, tt := range tests {
		tt := tt // Re-initializing variable so it is not changed while executing the closure below
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			abs, err := AbsURLPath(tt.path)
			if diff := cmp.Diff(tt.expected, err, cmpopts.EquateErrors()); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(tt.abs, abs); diff != "" {
				t.Fatalf("unexpected err (-want +got): \n%s", diff)
			}
		})
	}
}
