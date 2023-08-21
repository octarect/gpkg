package gpkg

import (
	"path"
)

type Config struct {
	CachePath string  `mapstructure:"cache_path"`
	Specs     []*Spec `mapstructure:"packages"`
}

func (c *Config) GetPackagesPath() string {
	return path.Join(c.CachePath, "packages")
}
