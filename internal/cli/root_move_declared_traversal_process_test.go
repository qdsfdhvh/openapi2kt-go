package cli_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRootMoveMultipleOwnersNativeDeclaredTraversal(t *testing.T) {
	for _, which := range []string{"source", "peer"} {
		for _, selection := range [][]string{{"--move-owner=a", "--move-owner=b"}, {"--spec=a"}, nil} {
			base, cfg, src, dst, _ := multipleMoveFixture(t, "unlocked", false)
			before := snapshot(t, src)
			peer := snapshot(t, filepath.Join(base, "unrelated"))
			text := string(read(t, cfg))
			old := "output_dir='generated'"
			name := "generated"
			if which == "peer" {
				old = "output_dir='unrelated'"
				name = "unrelated"
			}
			raw := filepath.ToSlash(base) + "/unused/../" + name
			changed := strings.Replace(text, old, "output_dir='"+raw+"'", 1)
			if changed == text {
				t.Fatal("test did not change declared root", which)
			}
			put(t, cfg, []byte(changed))
			args := []string{"generate", "--config", cfg, "--move-output-from", src, "--output", dst, "--diagnostic-format=json"}
			args = append(args, selection...)
			for _, cmd := range [][]string{append(append([]string{}, args...), "--check"), lifecycleValidate(args), args} {
				fails(t, invoke(t, base, cmd...), 4, "E_OUTPUT_PATH")
				if !reflect.DeepEqual(before, snapshot(t, src)) || !reflect.DeepEqual(peer, snapshot(t, filepath.Join(base, "unrelated"))) {
					t.Fatal("declared traversal refusal changed output", which, selection)
				}
				if _, err := os.Lstat(dst); !os.IsNotExist(err) {
					t.Fatal("declared traversal created target", err)
				}
			}
		}
	}
}
