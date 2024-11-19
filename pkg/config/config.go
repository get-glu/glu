package config

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path"
	"reflect"

	"github.com/get-glu/glu/internal/config"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Log         Log         `glu:"log"`
	Credentials Credentials `glu:"credentials"`
	Sources     struct {
		Git GitRepositories `glu:"git"`
		OCI OCIRepositories `glu:"oci"`
	} `glu:"sources"`
	Server Server `glu:"server"`
}

type defaulter interface {
	setDefaults() error
}

func (c *Config) SetDefaults() error {
	return processValue(reflect.ValueOf(c).Elem(), func(d defaulter) error {
		return d.setDefaults()
	})
}

type validater interface {
	validate() error
}

func (c *Config) Validate() error {
	return processValue(reflect.ValueOf(c).Elem(), func(v validater) error {
		return v.validate()
	})
}

func processValue[T any](val reflect.Value, method func(T) error) error {
	switch val.Kind() {
	case reflect.Struct:
		// Need to get addressable value for pointer receiver methods
		if val.CanAddr() {
			if impl, ok := val.Addr().Interface().(T); ok {
				if err := method(impl); err != nil {
					return err
				}
			}
		}

		// Recursively process only struct fields
		for i := 0; i < val.NumField(); i++ {
			if err := processValue(val.Field(i), method); err != nil {
				return err
			}
		}

	case reflect.Ptr:
		if !val.IsNil() {
			// Try to use the pointer directly first
			if impl, ok := val.Interface().(T); ok {
				if err := method(impl); err != nil {
					return err
				}
			}
			// Then recurse into the element for both structs and maps
			elemKind := val.Elem().Kind()
			if elemKind == reflect.Struct || elemKind == reflect.Map {
				return processValue(val.Elem(), method)
			}
		}
	case reflect.Map:
		if !val.IsNil() {
			if impl, ok := val.Addr().Interface().(T); ok {
				if err := method(impl); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func ReadFromPath(configPath string) (_ *Config, err error) {
	encoding := json.Unmarshal
	switch path.Ext(configPath) {
	case ".yaml", ".yml":
		encoding = yaml.Unmarshal
	}

	var fi io.ReadCloser
	fi, err = os.Open(configPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}

		slog.Warn("could not locate glu.yaml")

		fi = io.NopCloser(&nopReadCloser{})
	}

	defer fi.Close()

	decoder := config.NewDecoder[Config](fi, encoding)

	var conf Config
	if err := decoder.Decode(&conf); err != nil {
		return nil, err
	}

	if err := conf.SetDefaults(); err != nil {
		return nil, err
	}

	if err := conf.Validate(); err != nil {
		return nil, err
	}

	return &conf, nil
}

type nopReadCloser struct {
}

func (n nopReadCloser) Read(p []byte) (_ int, err error) {
	return 0, io.EOF
}
