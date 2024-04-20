package logger //nolint:testpackage

import (
	"testing"
)

func TestTrimPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "no dir",
			path: "main.go",
			want: "main.go",
		},
		{
			name: "one dir",
			path: "server/main.go",
			want: "server/main.go",
		},
		{
			name: "two dir",
			path: "cmd/server/main.go",
			want: "cmd/server/main.go",
		},
		{
			name: "three dir",
			path: "wb/dcim-backend/cmd/server/main.go",
			want: "cmd/server/main.go",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := TrimPath(tt.path); got != tt.want {
				t.Errorf("TrimPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
