package smis

import "testing"

func TestExtractPath(t *testing.T) {
	tests := []struct {
		name  string
		given string
		want  string
	}{
		{
			name:  "no slashes",
			given: "ping",
			want:  "",
		},
		{
			name:  "slash only",
			given: "/",
			want:  "",
		},
		{
			name:  "one slash at beginning",
			given: "/ping",
			want:  "/ping",
		},
		{
			name:  "two slashes",
			given: "/ping/something",
			want:  "/ping/something",
		},
		{
			name:  "two slashes with parameter",
			given: "/ping/:id",
			want:  "/ping",
		},
		{
			name:  "two slashes with ending slash",
			given: "/Pong/",
			want:  "/Pong",
		},
		{
			name:  "mixed cases",
			given: "/pingPong",
			want:  "/pingPong",
		},
		{
			name:  "with dash",
			given: "/ping-pong",
			want:  "/ping-pong",
		},
		{
			name:  "with underscore",
			given: "/ping_pong",
			want:  "/ping_pong",
		},
		{
			name:  "with digits",
			given: "/ping123",
			want:  "/ping123",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := extractPath(test.given)
			if test.want != actual {
				t.Errorf("expected %s but got %s", test.want, actual)
			}
		})
	}
}
