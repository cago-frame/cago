package code

import "github.com/cago-frame/cago/pkg/i18n"

func init() {
	i18n.Register(i18n.DefaultLang, zhCN)
}

var zhCN = map[int]string{
	UserIsBanned:          "用户已被禁用",
	UserNotFound:          "用户不存在",
	UserNotLogin:          "用户未登录",
	UsernameAlreadyExists: "用户名已存在",
}
