package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	DB       DBConfig       `mapstructure:"db"`
	Schedule ScheduleConfig `mapstructure:"schedule"`
	Admin    AdminConfig    `mapstructure:"admin"`
	PDF      PDFConfig      `mapstructure:"pdf"`
}

type ServerConfig struct {
	Addr         string        `mapstructure:"addr"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type DBConfig struct {
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
	User    string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name    string `mapstructure:"name"`
	SSLMode string `mapstructure:"sslmode"`
}

type ScheduleConfig struct {
	SemesterStartDate string `mapstructure:"semester_start_date"`
}

type AdminConfig struct {
	APIKey string `mapstructure:"api_key"`
}

type PDFConfig struct {
	ChromeExecutablePath string        `mapstructure:"chrome_executable_path"`
	Timeout              time.Duration `mapstructure:"timeout"`
}

type LoadOptions struct {
	ConfigPath string
}

func Load(opts LoadOptions) (*Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")

	if opts.ConfigPath != "" {
		v.SetConfigFile(opts.ConfigPath)
	} else {
		v.SetConfigName("config")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
	}

	v.SetEnvPrefix("ISPO")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if cfg.Server.Addr == "" {
		cfg.Server.Addr = "127.0.0.1:8080"
	}
	if cfg.DB.SSLMode == "" {
		cfg.DB.SSLMode = "disable"
	}
	if cfg.PDF.Timeout == 0 {
		cfg.PDF.Timeout = 20 * time.Second
	}

	return &cfg, nil
}
