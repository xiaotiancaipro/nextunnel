package configs

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	sharedtimezone "github.com/xiaotiancaipro/nextunnel/internal/shared/timezone"
)

func Load[T any](value T, file string) (*T, error) {

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil, err
	}

	var configs T
	if _, err := toml.DecodeFile(file, &configs); err != nil {
		return nil, err
	}

	if err := sharedtimezone.Init(new(Timezone).NameOrDefault()); err != nil {
		return nil, fmt.Errorf("invalid timezone config: %w", err)
	}

	return &configs, nil

}
