package testdata

import (
	"github.com/cago-frame/cago/pkg/utils/httputils"
	"github.com/cago-frame/cago/server/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Item struct {
	ID int64 `json:"id"`
}

// ListRequest 获取脚本列表
type ListRequest struct {
	mux.Meta              `path:"/script" method:"GET"`
	httputils.PageRequest `form:",inline"`
	Type                  int                     `form:"type" binding:"oneof=1 2 3 4"` // 1: 脚本 2: 库 3: 后台脚本 4: 定时脚本
	Sort                  string                  `form:"sort" binding:"oneof=today_download total_download score createtime updatetime"`
	PBool                 *bool                   `json:"p_bool"`
	ID                    primitive.ObjectID      `json:"id"`
	Data                  map[string][]*PkgStruct `json:"data"`
	Any                   any                     `json:"any"`
}
