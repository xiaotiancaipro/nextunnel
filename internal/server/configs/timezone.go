package configs

import sharedconfigs "github.com/xiaotiancaipro/nextunnel/internal/shared/configs"

const defaultTimezone = "Asia/Shanghai"

func (c *Configs) CheckTimezone() error {
	if c.Timezone == nil {
		c.Timezone = &sharedconfigs.Timezone{}
	}
	if c.Timezone.Location == "" {
		c.Timezone.Location = defaultTimezone
	}
	return nil
}
