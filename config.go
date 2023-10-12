package gpkg

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"

	"github.com/mitchellh/mapstructure"
)

type Config struct {
	CachePath string        `json:"cache_path"`
	Specs     []PackageSpec `json:"packages"`
}

func (c *Config) GetPackagesPath() string {
	return path.Join(c.CachePath, "packages")
}

type PackageSpec interface {
	Common() *CommonSpec
	Validate() error
	DisplayName() string
	PackagePath() string
}

type CommonSpec struct {
	From string `json:"from"`
	Pick string `json:"pick,omitempty"`
	Ref  string `json:"ref,omitempty"`
	ID   string `json:"id,omitempty"`

	config *Config
}

func (s *CommonSpec) Common() *CommonSpec {
	return s
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
	Repo string `json:"repo"`
}

func (s *GitHubReleaseSpec) Validate() error {
	return nil
}

func (s *GitHubReleaseSpec) DisplayName() string {
	if s.Common().ID != "" {
		return s.Common().ID
	} else {
		return s.Repo
	}
}

func (s *GitHubReleaseSpec) PackagePath() string {
	dir := strings.Replace(s.Repo, "/", "---", -1)
	return filepath.Join(s.config.GetPackagesPath(), dir)
}


func SpecEqual(a, b PackageSpec) bool {
	return a.PackagePath() == b.PackagePath()
}

func DecoderConfigOption(cfg *Config) func(*mapstructure.DecoderConfig) {
	return func(dc *mapstructure.DecoderConfig) {
		dc.TagName = "json"
		dc.DecodeHook = func(
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
			cs.config = cfg

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
}

//go:embed templates
var tmplFS embed.FS

func CreateConfigFile(cfgPath string) error {
	err := os.MkdirAll(filepath.Dir(cfgPath), 0744)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(cfgPath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	tmpl, _ := template.ParseFS(tmplFS, "templates/new_config.toml.tmpl")
	if err = tmpl.Execute(f, "dummy"); err != nil {
		return fmt.Errorf("failed to create a new config file from template. path=%q, err=%v", cfgPath, err)
	}

	return nil
}
