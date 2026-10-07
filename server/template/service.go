package template

// Service 是模板引擎（cube create）的入口。
// 默认模板来源存 settings.json 的 create 节（templateSource），直读不缓存。
type Service struct {
	tplSourceRoot string
}

func NewService(tplSourceRoot string) *Service {
	return &Service{
		tplSourceRoot: tplSourceRoot,
	}
}

// Create 生成项目到 targetPath。
func (s *Service) Create(sourceName string, tplName string, targetPath string, cliVars map[string]string) error {
	source, err := LoadSource(sourceName, s.tplSourceRoot)
	if err != nil {
		return err
	}

	return Create(source, tplName, targetPath, cliVars)
}
