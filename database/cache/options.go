package cache

import (
	cache2 "github.com/cago-frame/cago/database/cache/cache"
)

var (
	Expiration = cache2.Expiration
	WithDepend = cache2.WithDepend
)

func IsNil(err error) bool {
	return cache2.IsNil(err)
}
