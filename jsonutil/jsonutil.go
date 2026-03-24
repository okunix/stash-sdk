package jsonutil

import (
	"encoding/json"
	"io"
)

func Read[T any](r io.Reader) (T, error) {
	var dest T
	err := json.NewDecoder(r).Decode(&dest)
	return dest, err
}
