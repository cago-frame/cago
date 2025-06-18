package gen

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/cago-frame/cago/configs"
	"github.com/cago-frame/cago/database/db"
	"github.com/cago-frame/cago/internal/cmd/gen/utils"
	"github.com/spf13/cobra"
)

const entityTpl = `package {TableName}_entity

type {EntityName} struct {
{EntityField}
}
`

const repositoryTpl = `package {TableName}_repo

import (
	"context"

	"github.com/cago-frame/cago/database/db"
	"github.com/cago-frame/cago/pkg/consts"
	"{PkgName}/{TableName}_entity"
	"github.com/cago-frame/cago/pkg/utils/httputils"
)

type {Name}Repo interface {
	Find(ctx context.Context, id int64) (*{TableName}_entity.{Name}, error)
	FindPage(ctx context.Context, page httputils.PageRequest) ([]*{TableName}_entity.{Name}, int64, error)
	Create(ctx context.Context, {LowerName} *{TableName}_entity.{Name}) error
	Update(ctx context.Context, {LowerName} *{TableName}_entity.{Name}) error
	Delete(ctx context.Context, id int64) error
}

var default{Name} {Name}Repo

func {Name}() {Name}Repo {
	return default{Name}
}

func Register{Name}(i {Name}Repo) {
	default{Name} = i
}

type {LowerName}Repo struct {
}

func New{Name}() {Name}Repo {
	return &{LowerName}Repo{}
}

func (u *{LowerName}Repo) Find(ctx context.Context, id int64) (*{TableName}_entity.{Name}, error) {
	ret := &{TableName}_entity.{Name}{}
	if err := db.Ctx(ctx).Where("id=? and status=?", id, consts.ACTIVE).First(ret).Error; err != nil {
		if db.RecordNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return ret, nil
}

func (u *{LowerName}Repo) Create(ctx context.Context, {LowerName} *{TableName}_entity.{Name}) error {
	return db.Ctx(ctx).Create({LowerName}).Error
}

func (u *{LowerName}Repo) Update(ctx context.Context, {LowerName} *{TableName}_entity.{Name}) error {
	return db.Ctx(ctx).Updates({LowerName}).Error
}

func (u *{LowerName}Repo) Delete(ctx context.Context, id int64) error {
	return db.Ctx(ctx).Model(&{TableName}_entity.{Name}{}).Where("id=?", id).Update("status", consts.DELETE).Error
}

func (u *{LowerName}Repo) FindPage(ctx context.Context, page httputils.PageRequest) ([]*{TableName}_entity.{Name}, int64, error) {
	var list []*{TableName}_entity.{Name}
	var count int64
	find := db.Ctx(ctx).Model(&{TableName}_entity.{Name}{}).Where("status=?", consts.ACTIVE)
	if err := find.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := find.Order("createtime desc").Offset(page.GetOffset()).Limit(page.GetLimit()).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, count, nil
}

`

type Column struct {
	Field   string
	Type    string
	Null    string
	Key     string
	Default string
	Extra   string
}

type Index struct {
	Table      string
	NonUnique  int    `gorm:"column:Non_unique"`
	KeyName    string `gorm:"column:Key_name"`
	SeqInIndex int    `gorm:"column:Seq_in_index"`
	ColumnName string `gorm:"column:Column_name"`
}

func (c *Cmd) genDB(cmd *cobra.Command, args []string) error {
	table := args[0]
	// 读取appName
	abs, _ := filepath.Abs(".")
	cfg, err := configs.NewConfig(path.Base(abs))
	if err != nil {
		return err
	}
	c.pkgPath, c.pkgName, err = utils.FindRootPkgName("./")
	if err != nil {
		return err
	}
	dbComponent := db.Database()
	if err := dbComponent.Start(context.Background(), cfg); err != nil {
		return err
	}
	column := make([]Column, 0)
	if err := db.Default().
		Raw("describe " + db.Default().Config.NamingStrategy.TableName(table)).Scan(&column).Error; err != nil {
		return err
	}
	index := make([]Index, 0)
	if err := db.Default().
		Raw("show index from " + db.Default().Config.NamingStrategy.TableName(table)).Scan(&index).Error; err != nil {
		return err
	}
	if err := c.genEntity(table, column, index); err != nil {
		return err
	}
	// 生成仓库实现
	return c.genRepository(table)
}

// 获取当前包名
func (c *Cmd) getCurrentPkgName(dir string) (string, error) {
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	absPkgPath, err := filepath.Abs(c.pkgPath)
	if err != nil {
		return "", err
	}
	return c.pkgName + strings.TrimPrefix(absPath, absPkgPath), nil
}

func (c *Cmd) genRepository(table string) error {
	filepath := "internal/repository/" + table + "_repo/" + table + ".go"
	// 存在不创建
	if _, err := os.Stat(filepath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	// 获取表的索引信息
	repository := repositoryTpl
	pkgName, err := c.getCurrentPkgName("./internal/model/entity")
	if err != nil {
		return err
	}
	repository = strings.ReplaceAll(repository, "{PkgName}", pkgName)
	repository = strings.ReplaceAll(repository, "{Name}", utils.ToCamel(table))
	repository = strings.ReplaceAll(repository, "{LowerName}", utils.LowerFirstChar(utils.ToCamel(table)))
	repository = strings.ReplaceAll(repository, "{TableName}", table)
	// 写文件
	if err := os.MkdirAll("internal/repository/"+table+"_repo/", 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath, []byte(repository), 0644)
}

func (c *Cmd) genEntity(table string, column []Column, index []Index) error {
	filepath := "internal/model/entity/" + table + "_entity/" + table + ".go"
	// 存在不创建
	if _, err := os.Stat(filepath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	// 获取表的索引信息
	entity := entityTpl
	entity = strings.ReplaceAll(entity, "{TableName}", table)
	entity = strings.ReplaceAll(entity, "{EntityName}", utils.ToCamel(table))
	var entityField string
	for _, v := range column {
		entityField += "\t" + utils.ToCamel(v.Field) + " " + convSQLType(v.Type) + " `" + convSQLTag(v, index) + "`\n"
	}
	entity = strings.ReplaceAll(entity, "{EntityField}", strings.TrimRight(entityField, "\n"))
	// 写文件
	if err := os.MkdirAll("internal/model/entity/"+table+"_entity", 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath, []byte(entity), 0644)
}

func convSQLTag(column Column, index []Index) string {
	tags := make([]string, 0)
	tags = append(tags, "column:"+column.Field)
	tags = append(tags, "type:"+column.Type)
	if column.Default != "" {
		tags = append(tags, "default:"+column.Default)
	}
	if column.Null == "NO" {
		tags = append(tags, "not null")
	}
	if column.Key == "PRI" {
		tags = append(tags, "primary_key")
	} else if column.Key != "" {
		// 生成索引
		for _, v := range index {
			if v.ColumnName == column.Field {
				tag := "index:" + v.KeyName
				if v.NonUnique == 0 {
					tag += ",unique"
				}
				tags = append(tags, tag)
			}
		}
	}
	return "gorm:\"" + strings.Join(tags, ";") + "\""
}

func convSQLType(sqlType string) string {
	// 取括号前的类型
	if strings.Contains(sqlType, "(") {
		sqlType = sqlType[:strings.Index(sqlType, "(")]
	}
	switch sqlType {
	case "int", "mediumint", "bigint":
		return "int64"
	case "tinyint", "smallint", "bit":
		return "int32"
	case "float", "double":
		return "float64"
	case "varchar", "char", "text", "mediumtext", "longtext":
		return "string"
	case "datetime", "timestamp":
		return "time.Time"
	}
	return "interface{}"
}
