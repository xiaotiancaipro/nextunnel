package configs

import (
	"os"

	"github.com/BurntSushi/toml"
)

func Load[T any](value T, file string) (*T, error) {

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil, err
	}

	var configs T
	if _, err := toml.DecodeFile(file, &configs); err != nil {
		return nil, err
	}

	return &configs, nil

}
