package json

import (
	"encoding/json"
)

type Encoder struct{}

func NewJSONEncoder() *Encoder {
	return &Encoder{}
}

func (e *Encoder) Encode(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (e *Encoder) Decode(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func (e *Encoder) ContentType() string {
	return "application/json"
}
