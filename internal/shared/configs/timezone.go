package configs

const defaultTimezone = "Asia/Shanghai"

type Timezone struct {
	Location string `toml:"location"`
}

func (t *Timezone) NameOrDefault() string {
	if t != nil && t.Location != "" {
		return t.Location
	}
	return defaultTimezone
}
