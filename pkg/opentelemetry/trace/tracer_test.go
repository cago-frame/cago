package trace

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNewWithConfig_Empty(t *testing.T) {
	Convey("empty 类型：生成有效 trace_id 但不上报", t, func() {
		tp, err := NewWithConfig(context.Background(), &Config{Type: "empty"})
		So(err, ShouldBeNil)
		So(tp, ShouldNotBeNil)

		tracer := tp.Tracer("test")
		_, span := tracer.Start(context.Background(), "op")
		defer span.End()

		sc := span.SpanContext()
		So(sc.IsValid(), ShouldBeTrue)
		So(sc.TraceID().IsValid(), ShouldBeTrue)
		So(sc.SpanID().IsValid(), ShouldBeTrue)
	})

	Convey("noop 类型：SpanContext 无效", t, func() {
		tp, err := NewWithConfig(context.Background(), &Config{Type: "noop"})
		So(err, ShouldBeNil)
		So(tp, ShouldNotBeNil)

		tracer := tp.Tracer("test")
		_, span := tracer.Start(context.Background(), "op")
		defer span.End()

		So(span.SpanContext().IsValid(), ShouldBeFalse)
	})
}
