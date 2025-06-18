package minio

import (
	"errors"

	"github.com/cago-frame/cago/pkg/oss/oss"
	"github.com/minio/minio-go/v7"
)

type object struct {
	*minio.Object
}

func newObject(obj *minio.Object) oss.Object {
	return &object{obj}
}

type errStatObject struct {
	error
	statusCode int
}

func (e *errStatObject) StatusCode() int {
	return e.statusCode
}

func (o *object) Stat() (oss.ObjectInfo, error) {
	info, err := o.Object.Stat()
	if err != nil {
		var e minio.ErrorResponse
		if errors.As(err, &e) {
			return oss.ObjectInfo{}, &errStatObject{error: err, statusCode: e.StatusCode}
		}
		return oss.ObjectInfo{}, err
	}
	return oss.ObjectInfo{
		Key:          info.Key,
		ETag:         info.ETag,
		Size:         info.Size,
		LastModified: info.LastModified,
		ContentType:  info.ContentType,
	}, nil
}
