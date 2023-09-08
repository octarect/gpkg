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
	CachePath string  `mapstructure:"cache_path"`
	Specs     []ISpec `mapstructure:"packages"`
}

func (c *Config) GetPackagesPath() string {
	return path.Join(c.CachePath, "packages")
}

type ISpec interface {
	Common() *CommonSpec
	Source() Source
	Validate() error
	Init() error
	GetDirName() string
	OriginalMap() map[string]interface{}
}

type CommonSpec struct {
	From string
	Pick string
	Ref  string

	src Source
	m   map[string]interface{}
}

func (s *CommonSpec) Common() *CommonSpec {
	return s
}

func (s *CommonSpec) Source() Source {
	return s.src
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

type GitHubReleaseSpec struct {
	*CommonSpec
	Name string
	Pick string
}

func (s *GitHubReleaseSpec) Validate() error {
	return nil
}

func (s *GitHubReleaseSpec) Init() error {
	src, err := NewGitHubRelease(s.Name, s.Ref)
	if err != nil {
		return err
	}
	s.Common().src = src
	return nil
}

func (s *GitHubReleaseSpec) GetDirName() string {
	return strings.Replace(s.Name, "/", "---", -1)
}

func DecoderConfigOption(config *mapstructure.DecoderConfig) {
	config.DecodeHook = func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		if f.Kind() != reflect.Map || t != reflect.TypeOf((*ISpec)(nil)).Elem() {
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
