package installer

import "testing"

func TestParseRuntimeVersions(t *testing.T) {
	t.Run("lê node e npm", func(t *testing.T) {
		got, err := parseRuntimeVersions([]byte(`{"npm":"11.19.0","node":"24.21.0","v8":"13.6.233.17"}`))
		if err != nil || got.Node != "24.21.0" || got.Npm != "11.19.0" {
			t.Fatalf("got %+v, %v", got, err)
		}
	})

	for name, out := range map[string]string{
		"sem npm":       `{"node":"24.21.0"}`,
		"sem node":      `{"npm":"11.19.0"}`,
		"json inválido": `npm WARN config`,
	} {
		t.Run(name, func(t *testing.T) {
			if got, err := parseRuntimeVersions([]byte(out)); err == nil {
				t.Fatalf("esperava erro, got %+v", got)
			}
		})
	}
}

func TestNpmErrorSummary(t *testing.T) {
	tests := map[string]struct {
		output, want string
	}{
		"mantém só as linhas npm error": {
			output: "npm warn deprecated x\nnpm error code EEXIST\nnpm error path /bin/yarn\nnpm error EEXIST: file already exists\nnpm error extra\n",
			want:   "code EEXIST; path /bin/yarn; EEXIST: file already exists",
		},
		"sem linhas npm error devolve a saída": {
			output: "  falha inesperada\n",
			want:   "falha inesperada",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := npmErrorSummary([]byte(tt.output)); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
