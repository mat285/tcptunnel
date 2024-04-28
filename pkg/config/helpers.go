package config

import (
	"context"
	"os"

	"gopkg.in/yaml.v3"
)

type FileResolver struct {
	out   interface{}
	files []string
}

func (fr *FileResolver) Resolve(ctx context.Context) error {
	return ResolveFromFiles(fr.out, fr.files...)
}

func ReadYamlFile(file string, out interface{}) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, out)
}

func ResolveFromFiles(out interface{}, files ...string) error {
	for _, file := range files {
		err := ReadYamlFile(file, out)
		if err != nil {
			return err
		}
	}
	return nil
}
