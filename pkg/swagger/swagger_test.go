package swagger

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

const TestPath = "testdata"

func TestSwagger_parseSwagger(t *testing.T) {
	s := NewSwagger(TestPath)
	err := s.parseSwagger(path.Join(TestPath, "router.go"))
	assert.Nil(t, err)
	assert.Equalf(t, "api文档", s.swagger.Info.Title, "swagger title")
	assert.Equalf(t, "1.0", s.swagger.Info.Version, "swagger version")
	assert.Equalf(t, "/api/v1", s.swagger.BasePath, "swagger base path")
}

func TestSwagger_gen(t *testing.T) {
	s := NewSwagger(TestPath)
	err := s.gen()
	assert.Nil(t, err)
	paths := s.swagger.Paths.Paths
	assert.Equalf(t, 1, len(paths), "swagger path")
	//assert.Equal(t, "application/json", paths["/test/{uid}"].Get.Consumes[0])
}

func TestSwaggerGenWithoutGOPATHEnvironmentVariable(t *testing.T) {
	t.Setenv("GOPATH", "")

	s := NewSwagger(TestPath)
	assert.NoError(t, s.gen())
	assert.Len(t, s.swagger.Paths.Paths, 1, "swagger path")
}
