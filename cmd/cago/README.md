# Cago 脚手架

Cago有一个简单的脚手架，你可以使用脚手架快速开发你的项目

## 安装

```bash
go install github.com/cago-frame/cago/cmd/cago@latest
```

## 使用

你可以使用`cago -h`来查看脚手架支持的命令

### API

在`internal/api`目录下,定义好 api 请求结构,使用下面命令和自动生成`controller`代码和`swagger`文档

在`internal/service`目录下,定义好 service 接口,使用下面命令和自动生成`service`代码

```bash
cago gen
```
### 数据库模型

#### GORM

定义好表结构和`configs`文件后,使用下面命令和自动生成`model`代码和`repository`代码

```bash
cago gen gorm table_name
```

#### MongoDB

mongo数据库无需先定义数据库接口,使用下面命令即可直接生成`model`代码和`repository`代码

```bash
cago gen mongo table_name
```

