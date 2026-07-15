package fixapi

import "github.com/cago-frame/cago/server/mux"

type CreateRequest struct { // want "CAGO3004"
	mux.Meta `path:"users" method:"post"` // want "CAGO3005" "CAGO3006"
}
