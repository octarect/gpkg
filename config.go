package gpkg

import (
	"errors"
	"fmt"
	"path"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
)

type Config struct {
	CachePath string        `mapstructure:"cache_path"`
	Specs     []PackageSpec `mapstructure:"packages"`
}

func (c *Config) GetPackagesPath() string {
	return path.Join(c.CachePath, "packages")
}

type PackageSpec interface {
	Common() *CommonSpec
	Validate() error
	DisplayName() string
	DirName() string
	OriginalMap() map[string]interface{}
}

type CommonSpec struct {
	From string
	Pick string
	Ref  string
	ID   string

	m map[string]interface{}
}

func (s *CommonSpec) Common() *CommonSpec {
	return s
}

func (s *CommonSpec) OriginalMap() map[string]interface{} {
	return s.m
}

func (s *CommonSpec) Validate() error {
	if s.From == "" {
		return errors.New("from is required.")
	}
	return nil
}

func (s *CommonSpec) DisplayName() string {
	return s.ID
}

type GitHubReleaseSpec struct {
	*CommonSpec
	Name string
	Pick string
}

func (s *GitHubReleaseSpec) Validate() error {
	return nil
}

func (s *GitHubReleaseSpec) DirName() string {
	return strings.Replace(s.Name, "/", "---", -1)
}

func (s *GitHubReleaseSpec) DisplayName() string {
	if s.Common().ID != "" {
		return s.Common().ID
	} else {
		return s.Name
	}
}

func DecoderConfigOption(config *mapstructure.DecoderConfig) {
	config.DecodeHook = func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		if f.Kind() != reflect.Map || t != reflect.TypeOf((*PackageSpec)(nil)).Elem() {
			return data, nil
		}

		m, _ := data.(map[string]interface{})
		cs := &CommonSpec{}
		if err := mapstructure.Decode(m, &cs); err != nil {
			return nil, err
		}
		if err := cs.Validate(); err != nil {
			return nil, err
		}
		cs.m = m

		switch cs.From {
		case "ghr":
			ghr := &GitHubReleaseSpec{}
			if err := mapstructure.Decode(m, &ghr); err != nil {
				return nil, err
			}
			if err := ghr.Validate(); err != nil {
				return nil, err
			}
			ghr.CommonSpec = cs
			return ghr, nil
		default:
			return nil, fmt.Errorf("invalid spec. from=%s", cs.From)
		}
	}
}
