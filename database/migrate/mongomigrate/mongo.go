package mongomigrate

import (
	"context"
	"errors"
	"fmt"

	"github.com/cago-frame/cago/configs"

	"github.com/cago-frame/cago/database/mongo"
	"go.mongodb.org/mongo-driver/bson"
	mongo2 "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoMigrateTable struct {
	ID string `bson:"id"`
}

func (m *MongoMigrateTable) CollectionName() string {
	return "migrations"
}

type MongoMigrate struct {
	ctx        context.Context
	db         *mongo.Client
	migrations []*Migration
}

func New(ctx context.Context, mongo *mongo.Client, migrations []*Migration) *MongoMigrate {
	return &MongoMigrate{
		ctx:        ctx,
		db:         mongo,
		migrations: migrations,
	}
}

func (m *MongoMigrate) Migrate(option ...Option) error {
	opts := &Options{}
	for _, o := range option {
		o(opts)
	}
	// 获取所有的迁移记录
	collection := m.db.Database(m.ctx).Collection((&MongoMigrateTable{}).CollectionName())
	// 创建索引
	if _, err := collection.Indexes().CreateOne(mongo2.IndexModel{
		Keys:    bson.D{{Key: "id", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}
	m.db.Database(m.ctx)
	curs, err := collection.Find(bson.D{}, nil)
	if err != nil {
		return err
	}
	defer curs.Close(m.ctx)
	var records []*MongoMigrateTable
	if err := curs.All(m.ctx, &records); err != nil {
		return err
	}
	// 对比迁移记录和迁移函数
	if len(records) > len(m.migrations) {
		// 判断是否为pro并且拥有pre版本,那就忽略记录少的问题
		if !(opts.hasPre && configs.Default().Env == configs.PROD) {
			return errors.New("migrate records more than migrate functions")
		}
	}
	for n, record := range records {
		if record.ID != m.migrations[n].ID {
			return fmt.Errorf("migrate id not match: %s != %s", record.ID, m.migrations[n].ID)
		}
	}
	// 取出未迁移的函数
	migrations := m.migrations[len(records):]
	// 执行迁移
	for _, migration := range migrations {
		err := m.db.Client().UseSession(m.ctx, func(sessionContext mongo2.SessionContext) error {
			if err := sessionContext.StartTransaction(); err != nil {
				return err
			}
			if err := migration.Migrate(m.ctx, m.db); err != nil {
				return err
			}
			if _, err := collection.InsertOne(&MongoMigrateTable{ID: migration.ID}); err != nil {
				return err
			}
			return sessionContext.CommitTransaction(m.ctx)
		})
		if err != nil {
			return err
		}
	}
	return nil
}
