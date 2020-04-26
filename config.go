package usvc

import (
	"flag"
	"os"
	"time"
)

var (
	defaultConfig = Config{
		LogLevel:  "",
		LogFormat: "json",

		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,

		Addr: "",
		Port: "8080",
	}
)

type Config struct {
	// valid values are:
	// "", trace, debug, info, warn, error, fatal, panic
	LogLevel string
	// valid values are:
	// json, console
	LogFormat string

	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
	MaxHeaderBytes    int

	// Ex: 0.0.0.0
	Addr string
	// Ex: 8080
	Port string
}

// NewConfig returns a config with defaults
// overriden by options
func NewConfig(options ...ConfigOption) *Config {
	c := defaultConfig
	for _, o := range options {
		o(&c)
	}
	return &c
}

type ConfigOption func(c *Config)

// WithEnv overrides config values with ones retrieved from the environment
// LOG_LEVEL: LogLevel,
// LOG_FORMAT: LogFormat,
// PORT: Port,
func WithEnv() ConfigOption {
	return func(c *Config) {
		if e := os.Getenv("LOG_LEVEL"); e != "" {
			c.LogLevel = e
		}
		if e := os.Getenv("LOG_FORMAT"); e != "" {
			c.LogFormat = e
		}
		if e := os.Getenv("PORT"); e != "" {
			c.Port = e
		}
	}
}

// WithFlags registers flags with the provided flagset
// fs.Parse MUST be called before config is used
func WithFlags(fs *flag.FlagSet) ConfigOption {
	return func(c *Config) {
		fs.StringVar(&c.LogLevel, "log.level", defaultConfig.LogLevel, `levels: "", trace, debug, info, warn, error, fatal, panic`)
		fs.StringVar(&c.LogFormat, "log.format", defaultConfig.LogFormat, `format: json, console`)
		fs.StringVar(&c.Addr, "http.addr", defaultConfig.Addr, `address to listen on, ex 0.0.0.0`)
		fs.StringVar(&c.Port, "http.port", defaultConfig.Port, `port to listen on, ex 8080`)
	}
}

func WithLogLevel(level string) ConfigOption {
	return func(c *Config) {
		c.LogLevel = level
	}
}
func WithLogFormat(format string) ConfigOption {
	return func(c *Config) {
		c.LogFormat = format
	}
}
func WithReadTimeout(d time.Duration) ConfigOption {
	return func(c *Config) {
		c.ReadTimeout = d
	}
}
func WithReadHeaderTimeout(d time.Duration) ConfigOption {
	return func(c *Config) {
		c.ReadHeaderTimeout = d
	}
}
func WithWriteTimeout(d time.Duration) ConfigOption {
	return func(c *Config) {
		c.WriteTimeout = d
	}
}
func WithIdleTimeout(d time.Duration) ConfigOption {
	return func(c *Config) {
		c.IdleTimeout = d
	}
}
func WithShutdownTimeout(d time.Duration) ConfigOption {
	return func(c *Config) {
		c.ShutdownTimeout = d
	}
}
func WithMaxHeaderBytes(b int) ConfigOption {
	return func(c *Config) {
		c.MaxHeaderBytes = b
	}
}
func WithAddr(a string) ConfigOption {
	return func(c *Config) {
		c.Addr = a
	}
}
func WithPort(p string) ConfigOption {
	return func(c *Config) {
		c.Port = p
	}
}
