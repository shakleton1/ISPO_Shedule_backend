package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Log      LogConfig      `mapstructure:"log"`
	DB       DBConfig       `mapstructure:"db"`
	Schedule ScheduleConfig `mapstructure:"schedule"`
	Admin    AdminConfig    `mapstructure:"admin"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Push     PushConfig     `mapstructure:"push"`
	PDF      PDFConfig      `mapstructure:"pdf"`
}

type ServerConfig struct {
	Addr         string        `mapstructure:"addr"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Pretty bool   `mapstructure:"pretty"`
}

type DBConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	SSLMode  string `mapstructure:"sslmode"`
}

type ScheduleConfig struct {
	SemesterStartDate string `mapstructure:"semester_start_date"`
}

type AdminConfig struct {
	APIKey string `mapstructure:"api_key"`
}

type AuthConfig struct {
	JWTSecret              string        `mapstructure:"jwt_secret"`
	AccessTokenTTL         time.Duration `mapstructure:"access_token_ttl"`
	BootstrapAdminLogin    string        `mapstructure:"bootstrap_admin_login"`
	BootstrapAdminPassword string        `mapstructure:"bootstrap_admin_password"`
}

type PushConfig struct {
	Enabled bool      `mapstructure:"enabled"`
	FCM     FCMConfig `mapstructure:"fcm"`
}

type FCMConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	ProjectID       string        `mapstructure:"project_id"`
	CredentialsFile string        `mapstructure:"credentials_file"`
	Timeout         time.Duration `mapstructure:"timeout"`
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
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	if cfg.DB.SSLMode == "" {
		cfg.DB.SSLMode = "disable"
	}
	if cfg.PDF.Timeout == 0 {
		cfg.PDF.Timeout = 20 * time.Second
	}
	if cfg.Auth.AccessTokenTTL == 0 {
		cfg.Auth.AccessTokenTTL = 12 * time.Hour
	}
	if cfg.Push.FCM.Timeout == 0 {
		cfg.Push.FCM.Timeout = 5 * time.Second
	}

	return &cfg, nil
}
