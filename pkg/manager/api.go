package manager

type Api struct {
	m *Manager
}

func (api *Api) GetSetting() *Setting {
	return api.m.GetSetting()
}

func (api *Api) GetConfig() *Config {
	return api.m.GetConfig()
}

func (api *Api) SetConfig(cfg *Config) *Config {
	return api.m.SetConfig(cfg)
}

func (api *Api) Convert(paths []string) {
	api.m.Convert(paths)
}
