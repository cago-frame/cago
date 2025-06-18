package db

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/cago-frame/cago/configs"
	"github.com/cago-frame/cago/configs/memory"
	"github.com/stretchr/testify/assert"
	mysqlDriver "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type User struct {
	ID       int    `gorm:"primaryKey"`
	Username string `gorm:"column:username"`
}

type Info struct {
	ID     int    `gorm:"primaryKey"`
	Avatar string `gorm:"column:avatar"`
}

func TestDatabase(t *testing.T) {
	db1, mock1, err := sqlmock.New()
	assert.Nil(t, err)
	defer db1.Close()
	db2, mock2, err := sqlmock.New()
	assert.Nil(t, err)
	defer db2.Close()

	RegisterDriver("mock", func(config *Config) gorm.Dialector {
		if config.Dsn == "mock1" {
			return mysqlDriver.New(mysqlDriver.Config{SkipInitializeWithVersion: true, Conn: db1})
		}
		return mysqlDriver.New(mysqlDriver.Config{SkipInitializeWithVersion: true, Conn: db2})
	})
	cfg, _ := configs.NewConfig("test", configs.WithSource(
		memory.NewSource(map[string]interface{}{
			"env":   "dev",
			"debug": true,
			"dbs": GroupConfig{
				"default": &Config{
					Driver: "mock",
					Dsn:    "mock1",
				},
				"mock2": &Config{
					Driver: "mock",
					Dsn:    "mock2",
				},
			},
		}),
	))
	db := Database()
	err = db.Start(context.Background(), cfg)
	assert.Nil(t, err)
	mock1.ExpectQuery("SELECT").
		WithArgs(1, 1).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "username"}).AddRow(
				1, "admin"),
		)

	user := &User{ID: 1}
	Default().First(user)
	assert.Equal(t, 1, user.ID)
	assert.Equal(t, "admin", user.Username)

	// 测试mock2
	mock2.ExpectQuery("SELECT").
		WithArgs(2, 1).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "avatar"}).AddRow(
				2, "avatar"),
		)

	info := &Info{ID: 2}
	Use("mock2").First(info)
	assert.Equal(t, 2, info.ID)
	assert.Equal(t, "avatar", info.Avatar)
	mock1.ExpectQuery("SELECT").
		WithArgs(2, 1).
		WillReturnError(gorm.ErrRecordNotFound)
	err = Default().First(info).Error
	assert.Equal(t, gorm.ErrRecordNotFound, err)

	// 测试context
	ctx := context.Background()
	mock2.ExpectQuery("SELECT").
		WithArgs(3, 1).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "avatar"}).AddRow(
				3, "avatar3"),
		)
	mock1.ExpectQuery("SELECT").
		WithArgs(3, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	info = &Info{ID: 3}
	err = Ctx(ctx).First(info).Error
	assert.Equal(t, gorm.ErrRecordNotFound, err)
	CtxWith(ctx, "mock2").First(info)
	assert.Equal(t, 3, info.ID)
	assert.Equal(t, "avatar3", info.Avatar)

	ctx = WithContext(ctx, "mock2")
	mock2.ExpectQuery("SELECT").
		WithArgs(4, 1).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "avatar"}).AddRow(
				4, "avatar4"),
		)
	info = &Info{ID: 4}
	err = Ctx(ctx).First(info).Error
	assert.Nil(t, err)
	assert.Equal(t, 4, info.ID)
	assert.Equal(t, "avatar4", info.Avatar)

}
