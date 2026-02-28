package config

import (
	"fmt"
	"net"
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
	Addr                    string          `mapstructure:"addr"`
	ReadTimeout             time.Duration   `mapstructure:"read_timeout"`
	WriteTimeout            time.Duration   `mapstructure:"write_timeout"`
	ReadHeaderTimeout       time.Duration   `mapstructure:"read_header_timeout"`
	IdleTimeout             time.Duration   `mapstructure:"idle_timeout"`
	MaxHeaderBytes          int             `mapstructure:"max_header_bytes"`
	AdminImportMaxBodyBytes int64           `mapstructure:"admin_import_max_body_bytes"`
	CORS                    CORSConfig      `mapstructure:"cors"`
	Proxy                   ProxyConfig     `mapstructure:"proxy"`
	Debug                   DebugConfig     `mapstructure:"debug"`
	RateLimit               RateLimitConfig `mapstructure:"rate_limit"`
	Tracing                 TracingConfig   `mapstructure:"tracing"`
}

type ProxyConfig struct {
	// TrustedProxies is a list of trusted reverse proxy IPs or CIDRs.
	// If empty, X-Forwarded-For/X-Real-IP headers are ignored to prevent spoofing.
	TrustedProxies []string `mapstructure:"trusted_proxies"`
}

type DebugConfig struct {
	Pprof PprofConfig `mapstructure:"pprof"`
}

type PprofConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

type TracingConfig struct {
	Enabled          bool    `mapstructure:"enabled"`
	ServiceName      string  `mapstructure:"service_name"`
	OTLPHTTPEndpoint string  `mapstructure:"otlp_http_endpoint"`
	SampleRatio      float64 `mapstructure:"sample_ratio"`
}

type CORSConfig struct {
	// AllowedOrigins is a list of exact origins (e.g. "https://admin.example.com").
	// Use "*" to allow any origin (NOT compatible with AllowCredentials=true).
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	// AllowedMethods is a comma-separated list.
	AllowedMethods string `mapstructure:"allowed_methods"`
	// AllowedHeaders is a comma-separated list.
	AllowedHeaders string `mapstructure:"allowed_headers"`
	// ExposedHeaders is a comma-separated list.
	ExposedHeaders   string `mapstructure:"exposed_headers"`
	AllowCredentials bool   `mapstructure:"allow_credentials"`
}

type RateLimitConfig struct {
	Login       RateLimitRuleConfig `mapstructure:"login"`
	SchedulePDF RateLimitRuleConfig `mapstructure:"schedule_pdf"`
	AdminImport RateLimitRuleConfig `mapstructure:"admin_import"`
}

type RateLimitRuleConfig struct {
	Enabled bool    `mapstructure:"enabled"`
	RPS     float64 `mapstructure:"rps"`
	Burst   int     `mapstructure:"burst"`
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
	RefreshTokenTTL        time.Duration `mapstructure:"refresh_token_ttl"`
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
	if cfg.Server.ReadHeaderTimeout == 0 {
		cfg.Server.ReadHeaderTimeout = 5 * time.Second
	}
	if cfg.Server.IdleTimeout == 0 {
		cfg.Server.IdleTimeout = 60 * time.Second
	}
	if cfg.Server.MaxHeaderBytes == 0 {
		cfg.Server.MaxHeaderBytes = 1 << 20 // 1 MiB
	}
	if cfg.Server.AdminImportMaxBodyBytes == 0 {
		cfg.Server.AdminImportMaxBodyBytes = 25 << 20 // 25 MiB
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
	if cfg.Auth.RefreshTokenTTL == 0 {
		cfg.Auth.RefreshTokenTTL = 30 * 24 * time.Hour
	}
	if cfg.Push.FCM.Timeout == 0 {
		cfg.Push.FCM.Timeout = 5 * time.Second
	}

	if cfg.Server.Tracing.ServiceName == "" {
		cfg.Server.Tracing.ServiceName = "ispo-schedule"
	}
	if cfg.Server.Tracing.Enabled {
		if cfg.Server.Tracing.SampleRatio <= 0 {
			cfg.Server.Tracing.SampleRatio = 0.1
		}
	}

	for _, s := range cfg.Server.Proxy.TrustedProxies {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil, fmt.Errorf("server.proxy.trusted_proxies: empty entry")
		}
		if ip := net.ParseIP(s); ip != nil {
			continue
		}
		if _, _, err := net.ParseCIDR(s); err == nil {
			continue
		}
		return nil, fmt.Errorf("server.proxy.trusted_proxies: invalid entry %q (expected IP or CIDR)", s)
	}

	// Rate limit defaults (applied only when rule is enabled).
	if cfg.Server.RateLimit.Login.Enabled {
		if cfg.Server.RateLimit.Login.RPS <= 0 {
			cfg.Server.RateLimit.Login.RPS = 0.2 // ~12/min
		}
		if cfg.Server.RateLimit.Login.Burst <= 0 {
			cfg.Server.RateLimit.Login.Burst = 5
		}
	}
	if cfg.Server.RateLimit.SchedulePDF.Enabled {
		if cfg.Server.RateLimit.SchedulePDF.RPS <= 0 {
			cfg.Server.RateLimit.SchedulePDF.RPS = 0.1 // ~6/min
		}
		if cfg.Server.RateLimit.SchedulePDF.Burst <= 0 {
			cfg.Server.RateLimit.SchedulePDF.Burst = 2
		}
	}
	if cfg.Server.RateLimit.AdminImport.Enabled {
		if cfg.Server.RateLimit.AdminImport.RPS <= 0 {
			cfg.Server.RateLimit.AdminImport.RPS = 0.02 // ~1.2/min
		}
		if cfg.Server.RateLimit.AdminImport.Burst <= 0 {
			cfg.Server.RateLimit.AdminImport.Burst = 1
		}
	}

	return &cfg, nil
}
