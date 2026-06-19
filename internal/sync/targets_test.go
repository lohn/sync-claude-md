package sync

import (
	"reflect"
	"testing"
)

// TestResolveTargets verifies that the Claude/Gemini options map directly to
// the selected targets. The CLI decides defaults (CLAUDE.md unless --no-claude,
// GEMINI.md only with --gemini); resolveTargets is a literal mapping.
func TestResolveTargets(t *testing.T) {
	tests := []struct {
		name string
		opts Options
		want []target
	}{
		{
			name: "neither set selects nothing",
			opts: Options{},
			want: nil,
		},
		{
			name: "claude only",
			opts: Options{Claude: true},
			want: []target{claudeTarget},
		},
		{
			name: "gemini only",
			opts: Options{Gemini: true},
			want: []target{geminiTarget},
		},
		{
			name: "both set",
			opts: Options{Claude: true, Gemini: true},
			want: []target{claudeTarget, geminiTarget},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveTargets(tt.opts)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("resolveTargets(%+v) = %+v, want %+v", tt.opts, got, tt.want)
			}
		})
	}
}
