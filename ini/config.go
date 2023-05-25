package ini

import (
	"fmt"

	"github.com/go-ini/ini"
)

func SetCfgKey(cfg *ini.File, section, key, value string) error {
	if key == "" {
		return fmt.Errorf("no cfg key provided")
	}
	yes := cfg.Section(section).HasKey(key)
	if !yes {
		_, err := cfg.Section(section).NewKey(key, value)
		if err != nil {
			return err
		}
		return nil
	}
	cfg.Section(section).Key(key).SetValue(value)
	return nil
}
