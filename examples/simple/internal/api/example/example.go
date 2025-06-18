package example

import "github.com/cago-frame/cago/server/mux"

// PingRequest ping
type PingRequest struct {
	mux.Meta `path:"/example/ping" method:"GET"`
}

// PingResponse ping
type PingResponse struct {
	Pong string `json:"pong"`
}

// GinFunRequest gin function
type GinFunRequest struct {
	mux.Meta `path:"/example/gin-fun" method:"GET"`
}

// GinFunResponse gin function
type GinFunResponse struct {
}

// AuditRequest 审计操作
type AuditRequest struct {
	mux.Meta `path:"/example/audit" method:"POST"`
}

// AuditResponse 审计操作
type AuditResponse struct {
}
