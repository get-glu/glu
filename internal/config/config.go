package config

import (
	"io"
	"os"
	"strings"

	"github.com/mitchellh/mapstructure"
)

type Encoding func([]byte, any) error

type Decoder[C any] struct {
	rd      io.Reader
	enc     Encoding
	tagName string
}

func NewDecoder[C any](rd io.Reader, enc Encoding) *Decoder[C] {
	return &Decoder[C]{
		rd:      rd,
		enc:     enc,
		tagName: "glu",
	}
}

func (d *Decoder[C]) Decode(c *C) error {
	data, err := io.ReadAll(d.rd)
	if err != nil {
		return err
	}

	m := map[string]any{}
	if err := d.enc(data, m); err != nil {
		return err
	}

	d.parseEnvInto(m)

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: d.tagName,
	})

	if err != nil {
		return err
	}

	return decoder.Decode(c)
}

func (d *Decoder[C]) parseEnvInto(m map[string]any) {
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) < 2 {
			continue
		}

		key := strings.TrimPrefix(parts[0], strings.ToUpper(d.tagName+"_"))
		if parts[0] == key {
			// key does not have env prefix
			continue
		}

		addKV(m, strings.Split(strings.ToLower(key), "_"), parts[1])
	}
}

func addKV(m map[string]any, parts []string, value string) {
	if len(parts) == 0 {
		return
	}

	if len(parts) == 1 {
		v, ok := m[parts[0]]
		if !ok {
			m[parts[0]] = value
			return
		}

		switch v.(type) {
		case map[string]any:
			// we've already set a deaper structure for the key
		default:
			m[parts[0]] = value
			return
		}
		return
	}

	n, ok := m[parts[0]].(map[string]any)
	if !ok {
		n = map[string]any{}
	}

	addKV(n, parts[1:], value)

	m[parts[0]] = n
}
