package pose

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

// DecodeStrictJSON decodes exactly one JSON value from raw into v: an
// unknown field is refused, and so is anything after the value but white
// space, so a valid object followed by garbage or by a second value is
// malformed, not read as its first part (spec
// pose-strict-json-readers-require-the-end).
func DecodeStrictJSON(raw []byte, v any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return errors.New("content after the JSON value")
	}
	return nil
}
