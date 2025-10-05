package protocol

import (
	"errors"
	"strings"

	"github.com/Noooste/azuretls-api/internal/protocol/json"
)

var (
	ErrUnsupportedMediaType = errors.New("unsupported media type")
	ErrUnknownProtocol      = errors.New("unknown protocol")
)

type MessageEncoder interface {
	Encode(v any) ([]byte, error)
	Decode(data []byte, v any) error
	ContentType() string
}

func DetectProtocol(contentType string, data []byte) (MessageEncoder, error) {
	contentType = strings.ToLower(strings.TrimSpace(contentType))

	if contentType == "" {
		contentType = "application/json"
	}

	if strings.Contains(contentType, "application/json") {
		return json.NewJSONEncoder(), nil
	}

	return nil, ErrUnsupportedMediaType
}

func GetJSONEncoder() MessageEncoder {
	return json.NewJSONEncoder()
}

func IsJSONContentType(contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	return contentType == "" || strings.Contains(contentType, "application/json")
}
