package authn

import "golang.org/x/crypto/bcrypt"

type UserMetadata string

type User struct {
	// 用户ID
	ID string `json:"id"`
	// 用户名
	Username string `json:"username"`
	// 密码
	HashedPassword string `json:"-"`
	// 昵称
	Nickname string `json:"nickname,omitempty"`
	// 用户元数据
	Metadata map[UserMetadata]interface{} `json:"metadata,omitempty"`
}

func (u *User) CheckPassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.HashedPassword), []byte(password))
}

type RegisterRequest struct {
	// 用户名
	Username string `json:"username"`
	// 密码
	Password string `json:"password"`

	// 昵称
	Nickname string `json:"nickname,omitempty"`
	// 邮箱
	Email string `json:"email,omitempty"`
	// 手机号
	Phone string `json:"phone,omitempty"`
	// 用户元数据
	Metadata map[UserMetadata]interface{} `json:"metadata,omitempty"`
}

func (r *RegisterRequest) HashPassword() (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(r.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

type RegisterResponse struct {
	// 用户ID
	UserID string `json:"user_id"`
}

type UpdateUserRequest struct {
	// 用户名
	Username string `json:"username,omitempty"`
	// 密码
	Password string `json:"password,omitempty"`

	// 昵称
	Nickname string `json:"nickname,omitempty"`
	// 邮箱
	Email string `json:"email,omitempty"`
	// 手机号
	Phone string `json:"phone,omitempty"`
	// 用户元数据
	Metadata map[UserMetadata]interface{} `json:"metadata,omitempty"`
}
