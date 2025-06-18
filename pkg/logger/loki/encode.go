package loki

import (
	"encoding/json"

	"github.com/cago-frame/cago/pkg/utils"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

type lokiEncode struct {
	zapcore.Encoder
	labels map[string]interface{}
	pool   buffer.Pool
}

func NewLokiEncode(labels ...zap.Field) zapcore.Encoder {
	encodeConfig := zap.NewProductionEncoderConfig()
	encodeConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	ret := &lokiEncode{
		Encoder: zapcore.NewJSONEncoder(encodeConfig),
		pool:    buffer.NewPool(),
	}
	m := zapcore.NewMapObjectEncoder()
	for _, v := range labels {
		v.AddTo(m)
	}
	ret.labels = m.Fields
	return ret
}

func (e *lokiEncode) Clone() zapcore.Encoder {
	return e.clone()
}

func (e *lokiEncode) clone() *lokiEncode {
	return &lokiEncode{
		Encoder: e.Encoder.Clone(),
		pool:    e.pool,
		labels:  e.labels,
	}
}

type lokiPush struct {
	Streams [1]lokiPushStream `json:"streams"`
}

type lokiPushStream struct {
	Stream map[string]interface{} `json:"stream"`
	Values [1][2]string           `json:"values"`
}

func (e *lokiEncode) EncodeEntry(ent zapcore.Entry, fields []zap.Field) (*buffer.Buffer, error) {
	push := &lokiPush{
		Streams: [1]lokiPushStream{
			{
				Stream: e.labels,
				Values: [1][2]string{{
					utils.ToString(ent.Time.UnixNano()),
				}},
			},
		},
	}

	buf, err := e.Encoder.EncodeEntry(ent, fields)
	if err != nil {
		return nil, err
	}

	push.Streams[0].Values[0][1] = buf.String()

	b, err := json.Marshal(push)
	if err != nil {
		return nil, err
	}
	retBuf := e.pool.Get()
	if _, err := retBuf.Write(b); err != nil {
		return nil, err
	}
	return retBuf, nil
}
