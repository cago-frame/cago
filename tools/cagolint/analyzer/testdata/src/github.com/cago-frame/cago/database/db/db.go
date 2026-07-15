package db

import "context"

type DB struct{}

func Default() *DB { return &DB{} }
func Ctx(context.Context) *DB { return &DB{} }
