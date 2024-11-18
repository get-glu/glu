package config

import (
	"errors"
	"fmt"
)

const fieldErrFmt = "field %q: %w"

var (
	// errValidationRequired is returned when a required value is
	// either not supplied or supplied with empty value.
	errValidationRequired = errors.New("non-empty value is required")
	// errPositiveNonZeroValue is returned when a negative or zero value is provided.
	errPositiveNonZeroValue = errors.New("positive non-zero value required")
)

func errFieldWrap(field string, err error) error {
	return fmt.Errorf(fieldErrFmt, field, err)
}

func errFieldRequired(field string) error {
	return errFieldWrap(field, errValidationRequired)
}

func errFieldPositiveNonZero(field string) error {
	return errFieldWrap(field, errPositiveNonZeroValue)
}
