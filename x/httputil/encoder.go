package httputil

import (
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/x/binary"
)

// EncoderDecoder is an interface that extends binary.EncoderDecoder to
// add an HTTP content-type.
type EncoderDecoder interface {
	ContentType() string
	binary.EncoderDecoder
}

type typedEncoderDecoder struct {
	ct string
	binary.EncoderDecoder
}

func (t typedEncoderDecoder) ContentType() string { return t.ct }

var (
	JSONEncoderDecoder = typedEncoderDecoder{
		ct:             "application/json",
		EncoderDecoder: &binary.JSONEncoderDecoder{},
	}
	MsgPackEncoderDecoder = typedEncoderDecoder{
		ct:             "application/msgpack",
		EncoderDecoder: &binary.MsgPackEncoderDecoder{},
	}
)

var encoderDecoders = []EncoderDecoder{
	JSONEncoderDecoder,
	MsgPackEncoderDecoder,
}

func DetermineEncoderDecoder(contentType string) (EncoderDecoder, error) {
	for _, ecd := range encoderDecoders {
		if ecd.ContentType() == contentType {
			return ecd, nil
		}
	}
	return nil, errors.New("[encoding] - unable to determine encoding type")
}

func SupportedContentTypes() []string {
	var contentTypes []string
	for _, ecd := range encoderDecoders {
		contentTypes = append(contentTypes, ecd.ContentType())
	}
	return contentTypes
}
