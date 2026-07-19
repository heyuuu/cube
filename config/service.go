package config

type ConfigService struct {
	conf Config
}

func NewConfigService(conf Config) *ConfigService {
	return &ConfigService{
		conf: conf,
	}
}

func (s *ConfigService) Config() Config {
	return s.conf
}
