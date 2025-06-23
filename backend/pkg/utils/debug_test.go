package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFileAndLoC(t *testing.T) {
	type args struct {
		skip int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "hepi hepi hepi",
			args: args{
				skip: 0,
			},
			want: "watch-party/backend/pkg/utils/debug_test.go:28",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetFileAndLoC(tt.args.skip)
			assert.Equal(t, tt.want, got, "GetFileAndLoC() = %v, want %v", got, tt.want)
		})
	}
}
