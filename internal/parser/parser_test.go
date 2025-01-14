package parser

import (
	"bytes"
	"context"
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func FuzzParse(f *testing.F) {
	testdata := []string{
		"testdata/example.yaml",
	}

	for _, data := range testdata {
		b, _ := os.ReadFile(data)
		f.Add(b)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		_, err := parse(context.Background(), &decoder{rd: bytes.NewReader(data), enc: yaml.Unmarshal})
		if err != nil {
			// we only care about panics
			t.Skip()
		}
	})
}
