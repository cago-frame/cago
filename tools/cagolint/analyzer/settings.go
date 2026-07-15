package analyzer

// Settings configures Cago project conventions.
type Settings struct {
	APIDir        string `json:"api-dir"`
	ControllerDir string `json:"controller-dir"`
	ServiceDir    string `json:"service-dir"`
	RepositoryDir string `json:"repository-dir"`
	EntityDir     string `json:"entity-dir"`
}

func (s Settings) withDefaults() Settings {
	if s.APIDir == "" {
		s.APIDir = "/internal/api"
	}
	if s.ControllerDir == "" {
		s.ControllerDir = "/internal/controller/"
	}
	if s.ServiceDir == "" {
		s.ServiceDir = "/internal/service/"
	}
	if s.RepositoryDir == "" {
		s.RepositoryDir = "/internal/repository/"
	}
	if s.EntityDir == "" {
		s.EntityDir = "/internal/model/entity/"
	}
	return s
}
