package oss

import (
	"time"

	"github.com/cago-frame/cago"
	"github.com/cago-frame/cago/pkg/opentelemetry/trace"
	"github.com/cago-frame/cago/pkg/utils/wrap"
	"go.opentelemetry.io/otel/attribute"
	trace2 "go.opentelemetry.io/otel/trace"
)

const instrumName = "github.com/cago-frame/cago/pkg/oss"

func newWrap() *wrap.Wrap {
	w := wrap.New()
	if tp := trace.Default(); tp != nil {
		tracer := tp.Tracer(
			instrumName,
			trace2.WithInstrumentationVersion("semver:"+cago.Version()),
		)
		w.Wrap(func(ctx *wrap.Context) {
			sctx := ctx.Context
			sctx, span := tracer.Start(sctx, ctx.Name())
			defer span.End()
			switch ctx.Name() {
			case "PutObject":
				span.SetAttributes(attribute.String("objectName", ctx.Args(0).(string)))
			case "PreSignedPutObject":
				span.SetAttributes(attribute.String("objectName", ctx.Args(0).(string)))
				span.SetAttributes(attribute.Int64("expires", int64(ctx.Args(1).(time.Duration))))
			case "GetObject":
				span.SetAttributes(attribute.String("objectName", ctx.Args(0).(string)))
			case "PreSignedGetObject":
				span.SetAttributes(attribute.String("objectName", ctx.Args(0).(string)))
				span.SetAttributes(attribute.Int64("expires", int64(ctx.Args(1).(time.Duration))))
			case "RemoveObject":
				span.SetAttributes(attribute.String("objectName", ctx.Args(0).(string)))
			}
			ctx = ctx.WithContext(sctx)
			ctx.Next()
		})
	}
	return w
}
