package gen

import (
	"github.com/cago-frame/cago/internal/cmd/gen/utils"
	"github.com/spf13/cobra"
)

const mongoEntityTpl = `package {{.tableName}}_entity

import "go.mongodb.org/mongo-driver/bson/primitive"

type {{.entityName}} struct {
	ID         primitive.ObjectID ` + "`" + `bson:"_id"` + "`" + `
	Status     int8               ` + "`" + `bson:"status"` + "`" + `
	Createtime int64              ` + "`" + `bson:"createtime"` + "`" + `
	Updatetime int64              ` + "`" + `bson:"updatetime,omitempty"` + "`" + `
}

func ({{.firstChar}} *{{.entityName}}) CollectionName() string {
	return "{{.tableName}}"
}
`

const mongoRepositoryTpl = `package {{.tableName}}_repo

import (
	"context"

	"github.com/cago-frame/cago/database/mongo"
	"{{.pkgName}}/internal/model/entity/{{.tableName}}_entity"
	"github.com/cago-frame/cago/pkg/consts"
	"github.com/cago-frame/cago/pkg/utils/httputils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type {{.entityName}}Repo interface {
	Find(ctx context.Context, id primitive.ObjectID) (*{{.tableName}}_entity.{{.entityName}}, error)
	FindPage(ctx context.Context, page httputils.PageRequest) ([]*{{.tableName}}_entity.{{.entityName}}, int64, error)
	Create(ctx context.Context, {{.lowerName}} *{{.tableName}}_entity.{{.entityName}}) error
	Update(ctx context.Context, {{.lowerName}} *{{.tableName}}_entity.{{.entityName}}) error
	Delete(ctx context.Context, id primitive.ObjectID) error
}

var default{{.entityName}} {{.entityName}}Repo

func {{.entityName}}() {{.entityName}}Repo {
	return default{{.entityName}}
}

func Register{{.entityName}}(i {{.entityName}}Repo) {
	default{{.entityName}} = i
}

type {{.lowerName}}Repo struct {
}

func New{{.entityName}}() {{.entityName}}Repo {
	return &{{.lowerName}}Repo{}
}

func (u *{{.lowerName}}Repo) Find(ctx context.Context, id primitive.ObjectID) (*{{.tableName}}_entity.{{.entityName}}, error) {
	{{.lowerName}} := &{{.tableName}}_entity.{{.entityName}}{}
	err := mongo.Ctx(ctx).Collection({{.lowerName}}.CollectionName()).FindOne(bson.M{
		"_id":     id,
		"status": consts.ACTIVE,
	}).Decode({{.lowerName}})
	if err != nil {
		if mongo.IsNoDocuments(err) {
			return nil, nil
		}
		return nil, err
	}
	return {{.lowerName}}, nil
}


func (u *{{.lowerName}}Repo) Create(ctx context.Context, {{.lowerName}} *{{.tableName}}_entity.{{.entityName}}) error {
	{{.lowerName}}.ID = primitive.NewObjectID()
	_, err := mongo.Ctx(ctx).Collection({{.lowerName}}.CollectionName()).InsertOne({{.lowerName}})
	return err
}

func (u *{{.lowerName}}Repo) Update(ctx context.Context, {{.lowerName}} *{{.tableName}}_entity.{{.entityName}}) error {
	_, err := mongo.Ctx(ctx).Collection({{.lowerName}}.CollectionName()).UpdateOne(bson.M{
		"_id":    {{.lowerName}}.ID,
		"status": consts.ACTIVE,
	}, bson.M{
		"$set": {{.lowerName}},
	})
	return err
}

func (u *{{.lowerName}}Repo) Delete(ctx context.Context, id primitive.ObjectID) error {
	{{.lowerName}} := &{{.tableName}}_entity.{{.entityName}}{}
	_, err := mongo.Ctx(ctx).Collection({{.lowerName}}.CollectionName()).
		UpdateOne(bson.M{
			"_id":    id,
			"status": consts.ACTIVE,
		}, bson.M{
			"$set": bson.M{
				"status": consts.DELETE,
			},
		})
	return err
}

func (u *{{.lowerName}}Repo) FindPage(ctx context.Context, page httputils.PageRequest) ([]*{{.tableName}}_entity.{{.entityName}}, int64, error) {
	{{.lowerName}} := {{.tableName}}_entity.{{.entityName}}{}
	filter := bson.M{
		"status": consts.ACTIVE,
	}

	findOptions := options.Find()
	findOptions.SetSkip(int64(page.GetOffset()))
	findOptions.SetLimit(int64(page.GetSize()))
	findOptions.SetSort(bson.M{"createtime": -1})

	total, err := mongo.Ctx(ctx).Collection({{.lowerName}}.CollectionName()).CountDocuments(filter)
	if err != nil {
		return nil, 0, err
	}

	curs, err := mongo.Ctx(ctx).Collection({{.lowerName}}.CollectionName()).Find(filter, findOptions)
	if err != nil {
		return nil, 0, err
	}
	userList := make([]*{{.tableName}}_entity.{{.entityName}}, 0, page.GetSize())
	if err = curs.All(ctx, &userList); err != nil {
		return nil, 0, err
	}

	return userList, total, nil
}


`

func (c *Cmd) genMongo(cmd *cobra.Command, args []string) error {
	table := args[0]
	entityName := utils.UpperFirstChar(utils.ToCamel(table))
	filepath := "internal/model/entity/" + table + "_entity/" + table + ".go"
	f, err := utils.ParseTemplate(mongoEntityTpl, map[string]interface{}{
		"entityName": entityName,
		"tableName":  table,
		"firstChar":  table[:1],
	})
	if err != nil {
		return err
	}
	if err := utils.WriteFile(filepath, f); err != nil {
		return err
	}
	filepath = "internal/repository/" + table + "_repo/" + table + ".go"
	c.pkgPath, c.pkgName, err = utils.FindRootPkgName("./")
	if err != nil {
		return err
	}
	pkgName, err := c.getCurrentPkgName(".")
	if err != nil {
		return err
	}
	f, err = utils.ParseTemplate(mongoRepositoryTpl, map[string]interface{}{
		"entityName": entityName,
		"lowerName":  utils.LowerFirstChar(entityName),
		"tableName":  table,
		"firstChar":  table[:1],
		"pkgName":    pkgName,
	})
	if err != nil {
		return err
	}
	return utils.WriteFile(filepath, f)
}
