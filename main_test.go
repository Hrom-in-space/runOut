package main

import "testing"

func Test_parseNeedsArgs(t *testing.T) {
	tests := []struct {
		name    string
		argName string
		want    string
		wantErr bool
	}{
		{
			name:    "success",
			argName: "{\"name\":\"бананы\"}",
			// argName: `{"name": "test"}`,
			want:    "бананы",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseNeedsArgs(tt.argName)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseNeedsArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseNeedsArgs() got = %v, want %v", got, tt.want)
			}
		})
	}
}
