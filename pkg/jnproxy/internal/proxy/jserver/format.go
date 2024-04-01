package jserver

import (
	"encoding/json"
	"fmt"
	"time"
)

func format(content []byte) ([]byte, error) {
	type Content struct {
		Time  string `json:"time"`
		Value []byte `json:"content"`
	}
	c := Content{
		Time:  time.Now().UTC().Format(time.RFC3339),
		Value: content,
	}
	// TODO: decide the format we want. Is jupyter text-based only?
	// NOTE: https://golang.org/pkg/encoding/json/#Marshal
	// Array and slice values encode as JSON arrays, except that []byte encodes as a base64-encoded string, and a nil slice encodes as the null JSON object.
	// use base64.StdEncoding.DecodeString() for decoding.
	ret, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	return ret, nil
}
