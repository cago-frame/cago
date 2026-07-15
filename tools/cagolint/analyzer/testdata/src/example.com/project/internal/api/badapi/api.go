package badapi

import (
	_ "example.com/project/internal/service/user_svc" // want "CAGO2002"
	"github.com/cago-frame/cago/server/mux"
)

type GetUserRequest struct { // want "CAGO3004"
	mux.Meta `path:"users/:id" method:"fetch"` // want "CAGO3005" "CAGO3006" "CAGO3007"
}

type WrongName struct { // want "CAGO3003"
	mux.Meta `path:"/wrong" method:"GET"`
}

type ScalarRequest string // want "CAGO3001"

type MissingMetaRequest struct{} // want "CAGO3002"
