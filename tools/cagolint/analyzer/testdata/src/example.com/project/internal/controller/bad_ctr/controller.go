package bad_ctr

import (
	api "example.com/project/internal/api/badapi"
)

type Bad struct{}

func (b *Bad) Handle(ctx string, req *api.GetUserRequest) string { // want "CAGO4002" "CAGO4003" "CAGO4004"
	return ctx
}
