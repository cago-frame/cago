package user_repo

import (
	"context"
	"strconv"
	"time"

	"github.com/cago-frame/cago/pkg/iam/authn"

	"github.com/cago-frame/cago/database/db"
	"github.com/cago-frame/cago/examples/simple/internal/model/entity/user_entity"
	"github.com/cago-frame/cago/pkg/consts"
	"github.com/cago-frame/cago/pkg/utils/httputils"
)

//go:generate mockgen -source user.go -destination mock/user.go
type UserRepo interface {
	authn.Database

	Find(ctx context.Context, id int64) (*user_entity.User, error)
	FindPage(ctx context.Context, page httputils.PageRequest) ([]*user_entity.User, int64, error)
	Create(ctx context.Context, user *user_entity.User) error
	Update(ctx context.Context, user *user_entity.User) error
	Delete(ctx context.Context, id int64) error

	FindByUsername(ctx context.Context, username string) (*user_entity.User, error)
}

var defaultUser UserRepo

func User() UserRepo {
	return defaultUser
}

func RegisterUser(i UserRepo) {
	defaultUser = i
}

type userRepo struct {
}

func NewUser() UserRepo {
	return &userRepo{}
}

func (u *userRepo) Find(ctx context.Context, id int64) (*user_entity.User, error) {
	ret := &user_entity.User{}
	if err := db.Ctx(ctx).Where("id=? and status=?", id, consts.ACTIVE).First(ret).Error; err != nil {
		if db.RecordNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return ret, nil
}

func (u *userRepo) Create(ctx context.Context, user *user_entity.User) error {
	return db.Ctx(ctx).Create(user).Error
}

func (u *userRepo) Update(ctx context.Context, user *user_entity.User) error {
	return db.Ctx(ctx).Updates(user).Error
}

func (u *userRepo) Delete(ctx context.Context, id int64) error {
	return db.Ctx(ctx).Model(&user_entity.User{}).Where("id=?", id).Update("status", consts.DELETE).Error
}

func (u *userRepo) FindPage(ctx context.Context, page httputils.PageRequest) ([]*user_entity.User, int64, error) {
	var list []*user_entity.User
	var count int64
	find := db.Ctx(ctx).Model(&user_entity.User{}).Where("status=?", consts.ACTIVE)
	if err := find.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := find.Order("createtime desc").Offset(page.GetOffset()).Limit(page.GetLimit()).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, count, nil
}

func (u *userRepo) FindByUsername(ctx context.Context, username string) (*user_entity.User, error) {
	ret := &user_entity.User{}
	if err := db.Ctx(ctx).Where("username=?", username).First(ret).Error; err != nil {
		if db.RecordNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return ret, nil
}

func (u *userRepo) Register(ctx context.Context, req *authn.RegisterRequest) (*authn.RegisterResponse, error) {
	user := &user_entity.User{
		ID:         0,
		Username:   req.Username,
		Status:     consts.ACTIVE,
		Createtime: time.Now().Unix(),
		Updatetime: time.Now().Unix(),
	}
	var err error
	user.HashedPassword, err = req.HashPassword()
	if err != nil {
		return nil, err
	}
	if err := db.Ctx(ctx).Create(user).Error; err != nil {
		return nil, err
	}
	return &authn.RegisterResponse{
		UserID: strconv.FormatInt(user.ID, 10),
	}, nil
}

func (u *userRepo) GetUserByUsername(ctx context.Context, username string, opts ...authn.GetUserOption) (*authn.User, error) {
	return u.GetUserByWhere(ctx, map[string]interface{}{"username": username}, opts...)
}

func (u *userRepo) GetUserByID(ctx context.Context, userID string, opts ...authn.GetUserOption) (*authn.User, error) {
	return u.GetUserByWhere(ctx, map[string]interface{}{"id": userID}, opts...)
}

func (u *userRepo) GetUserByWhere(ctx context.Context, where map[string]interface{}, opts ...authn.GetUserOption) (*authn.User, error) {
	options := authn.NewGetUserOptions(opts...)
	user := &user_entity.User{}
	tx := db.Ctx(ctx).Where("status=?", consts.ACTIVE)
	for k, v := range where {
		switch k {
		case "username":
			tx = tx.Where("username like ?", v)
		case "id":
			tx = tx.Where("id=?", v)
		}
	}
	if err := tx.First(user).Error; err != nil {
		if db.RecordNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	ret := &authn.User{
		ID:             strconv.FormatInt(user.ID, 10),
		Username:       user.Username,
		HashedPassword: "",
		Nickname:       "",
		Metadata:       nil,
	}
	if options.WithPassword {
		ret.HashedPassword = user.HashedPassword
	}
	return ret, nil
}

//
//func (u *userRepo) UpdateUser(ctx context.Context, userID string, user *authn.UpdateUserRequest) error {
//	uid, err := strconv.ParseInt(userID, 10, 64)
//	if err != nil {
//		return err
//	}
//	user := &user_entity.User{
//		ID:       uid,
//		Username: user.Username,
//	}
//}
