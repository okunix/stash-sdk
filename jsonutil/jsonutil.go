package jsonutil

import (
	"encoding/json"
	"errors"
	"io"
)

func Read[T any](r io.Reader) (T, error) {
	var dest T
	err := json.NewDecoder(r).Decode(&dest)
	return dest, err
}

type Message struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Detail  any    `json:"detail"`
}

type ValidationProblems map[string]interface{}

func (p ValidationProblems) Error() string {
	errJson, _ := json.Marshal(p)
	return string(errJson)
}

func ForgeError(r io.Reader) error {
	errMsg := "error status code"
	msg, err := Read[Message](r)
	if err != nil {
		return errors.New(errMsg)
	}
	if msg.Message == "Validation Error" && msg.Detail != nil {
		return msg.Detail.(ValidationProblems)
	}
	if msg.Detail != nil {
		errMsg = msg.Detail.(string)
	} else {
		errMsg = msg.Message
	}
	return errors.New(errMsg)
}
