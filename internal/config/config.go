package config

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
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
	if err := d.enc(data, &m); err != nil {
		return fmt.Errorf("unmarshalling configuration: %w", err)
	}

	d.parseEnvInto(m)

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: d.tagName,
		Result:  c,
		MatchName: func(mapKey, fieldName string) bool {
			stripUnderscore := func(s string) string {
				return strings.ReplaceAll(s, "_", "")
			}

			return strings.EqualFold(stripUnderscore(mapKey), stripUnderscore(fieldName))
		},
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.DecodeHookFuncType(func(from, to reflect.Type, i interface{}) (interface{}, error) {
				if from.Kind() != reflect.String {
					return i, nil
				}

				if to.Kind() == reflect.Int {
					val, err := strconv.Atoi(i.(string))
					return val, err
				}

				if to.Kind() == reflect.Int64 {
					return strconv.ParseInt(i.(string), 10, 64)
				}

				return i, nil
			}),
		),
	})

	if err != nil {
		return fmt.Errorf("creating decoder: %w", err)
	}

	if err := decoder.Decode(m); err != nil {
		return fmt.Errorf("decoding configuration: %w", err)
	}

	return nil
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
