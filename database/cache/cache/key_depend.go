package cache

import (
	"context"
	"errors"
	"math/rand/v2"
)

var ErrDependNotValid = errors.New("depend not valid")

// KeyDepend key依赖
type KeyDepend struct {
	store Cache
	key   string
	value Int64DependValue
}

func NewKeyDepend(store Cache, key string) *KeyDepend {
	return &KeyDepend{
		store: store,
		key:   key,
	}
}

func WithKeyDepend(store Cache, key string) Option {
	return func(options *Options) {
		options.Depend = NewKeyDepend(store, key)
	}
}

// InvalidKey 使key失效
func (v *KeyDepend) InvalidKey(ctx context.Context) error {
	return v.store.Set(ctx, v.key, Int64DependValue(-1)).Err()
}

type Int64DependValue int64

func (i Int64DependValue) Equate(d DependValue) bool {
	v, ok := d.(Int64DependValue)
	return ok && i == v
}

// Val 获取依赖的值
func (v *KeyDepend) Val(ctx context.Context) (DependValue, error) {
	var i int64
	if err := v.store.Get(ctx, v.key).Scan(&i); err != nil || i == -1 {
		newValue := rand.Int64()
		v.store.Set(ctx, v.key, Int64DependValue(newValue))
		return Int64DependValue(newValue), nil //nolint:nilerr
	}
	return Int64DependValue(i), nil
}

func (v *KeyDepend) ValInterface() (DependValue, error) {
	return &v.value, nil
}

// Valid 检查依赖是否有效
func (v *KeyDepend) Valid(ctx context.Context) error {
	val, err := v.Val(ctx)
	if err != nil {
		return err
	}
	if val.Equate(v.value) {
		return nil
	}
	return ErrDependNotValid
}

// NilDep NilDepend 用于set的时候反序列化,减少一次dep判断,会跳过依赖检查
type NilDep struct {
	Depend
}

func (n *NilDep) ValInterface() (DependValue, error) {
	return nil, nil
}

func (n *NilDep) Valid(ctx context.Context) error {
	return nil
}
