package gpkg

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/mitchellh/mapstructure"
)

type State struct {
	Spec PackageSpec `json:"spec"`
	Path string      `json:"path"`
}

type StateData struct {
	States []State `json:"states"`
}

func DecodeStateData(r io.Reader) (*StateData, error) {
	sd := &StateData{}
	var raw map[string]interface{}
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return nil, fmt.Errorf("Error decoding json in a state file. err=%v", err)
	}

	o := &mapstructure.DecoderConfig{}
	o.Result = sd
	DecoderConfigOption(&Config{})(o)

	dec, _ := mapstructure.NewDecoder(o)
	if err := dec.Decode(raw); err != nil {
		return nil, fmt.Errorf("Error decoding json in a state file. err=%v", err)
	}

	return sd, nil
}

func LoadStateDataFromFile(path string) (*StateData, error) {
	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &StateData{}, nil
		} else {
			return nil, err
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	states, err := DecodeStateData(f)
	if err != nil {
		return nil, err
	}

	return states, nil
}

func (sd *StateData) Save(w io.Writer) error {
	bs, err := json.MarshalIndent(sd, "", "  ")
	if err != nil {
		return fmt.Errorf("Failed to encoding states to JSON. err=%v", err)
	}
	_, err = w.Write(bs)
	if err != nil {
		return fmt.Errorf("Failed to write a state file: err=%v", err)
	}
	return nil
}

func (sd *StateData) SaveToFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return sd.Save(f)
}

func (sd *StateData) FindState(spec PackageSpec) (int, *State, error) {
	var found *State
	idx := -1
	for i, st := range sd.States {
		if ok := SpecEqual(st.Spec, spec); ok {
			idx = i
			found = &st
		}
	}
	if idx == -1 {
		return -1, nil, fmt.Errorf("No state found. spec=%s", spec.DisplayName())
	}
	return idx, found, nil
}

func (sd *StateData) Upsert(spec PackageSpec, ref string) {
	idx, _, err := sd.FindState(spec)

	s0 := State{
		Spec: spec,
		Path: spec.PackagePath(),
	}

	if err != nil {
		// New package
		sd.States = append(sd.States, s0)
	} else {
		// Found
		sd.States[idx] = s0
	}
}
