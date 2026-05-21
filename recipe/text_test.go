package recipe

import (
	"strings"
	"testing"
)

func TestClassifyText(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		size int
		want string
	}{
		{name: "utf8 text", data: []byte("hello\n"), size: 6, want: TextKindText},
		{name: "nul binary", data: []byte{'h', 0, 'i'}, size: 3, want: TextKindBinary},
		{name: "invalid utf8", data: []byte{0xff, 0xfe}, size: 2, want: TextKindBinary},
		{name: "large", data: []byte("hello"), size: RecipeTextDiffMaxBytes + 1, want: TextKindLarge},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassifyText(tc.data, tc.size); got != tc.want {
				t.Fatalf("ClassifyText = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestUnifiedDiffNormalizesCRLF(t *testing.T) {
	diff := UnifiedTextDiff("old.txt", "new.txt", []byte("a\r\nb\r\n"), []byte("a\r\nc\r\n"))
	if !strings.Contains(diff, "-b") || !strings.Contains(diff, "+c") {
		t.Fatalf("diff missing expected lines:\n%s", diff)
	}
}
