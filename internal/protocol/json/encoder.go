package json

import (
	"encoding/json"
	"io"
)

type Encoder struct{}

func NewJSONEncoder() *Encoder {
	return &Encoder{}
}

func (e *Encoder) Encode(w io.Writer, v any) error {
	return json.NewEncoder(w).Encode(v)
}

func (e *Encoder) Decode(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}

func (e *Encoder) ContentType() string {
	return "application/json"
}
