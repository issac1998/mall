package config

import (
	"fmt"
	"time"
)

// Config represents the global configuration
type Config struct {
	Server       ServerConfig       `mapstructure:"server"`
	Database     DatabaseConfig     `mapstructure:"database"`
	Redis        RedisConfig        `mapstructure:"redis"`
	Etcd         EtcdConfig         `mapstructure:"etcd"`
	Queue        QueueConfig        `mapstructure:"queue"`
	Log          LogConfig          `mapstructure:"log"`
	Metrics      MetricsConfig      `mapstructure:"metrics"`
	Tracing      TracingConfig      `mapstructure:"tracing"`
	RateLimit    RateLimitConfig    `mapstructure:"rate_limit"`
	CircuitBreak CircuitBreakConfig `mapstructure:"circuit_break"`
	Cache        CacheConfig        `mapstructure:"cache"`
	Security     SecurityConfig     `mapstructure:"security"`
	Seckill      SeckillConfig      `mapstructure:"seckill"`
}

// ServerConfig represents HTTP server configuration
type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Mode         string        `mapstructure:"mode"` // debug, release, test
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
	MaxHeaderMB  int           `mapstructure:"max_header_mb"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Driver          string        `mapstructure:"driver"`
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"`
	DBName          string        `mapstructure:"dbname"`
	Charset         string        `mapstructure:"charset"`
	ParseTime       bool          `mapstructure:"parse_time"`
	Loc             string        `mapstructure:"loc"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
	LogLevel        string        `mapstructure:"log_level"` // silent, error, warn, info
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db"`
	PoolSize     int           `mapstructure:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns"`
	MaxRetries   int           `mapstructure:"max_retries"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	PoolTimeout  time.Duration `mapstructure:"pool_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

// EtcdConfig represents etcd configuration
type EtcdConfig struct {
	Endpoints   []string      `mapstructure:"endpoints"`
	Username    string        `mapstructure:"username"`
	Password    string        `mapstructure:"password"`
	DialTimeout time.Duration `mapstructure:"dial_timeout"`
	Namespace   string        `mapstructure:"namespace"`
}

// QueueConfig represents message queue configuration
type QueueConfig struct {
	Driver      string `mapstructure:"driver"` 
	Brokers     []string `mapstructure:"brokers"`
	Topic       string `mapstructure:"topic"`
	GroupID     string `mapstructure:"group_id"`
	Partitions  int    `mapstructure:"partitions"`
	Replication int    `mapstructure:"replication"`
}

// LogConfig represents logging configuration
type LogConfig struct {
	Level      string `mapstructure:"level"`      
	Format     string `mapstructure:"format"`    
	Output     string `mapstructure:"output"`    
	Filename   string `mapstructure:"filename"`  
	MaxSize    int    `mapstructure:"max_size"`   
	MaxAge     int    `mapstructure:"max_age"`    
	MaxBackups int    `mapstructure:"max_backups"`
	Compress   bool   `mapstructure:"compress"`
}

// MetricsConfig represents metrics configuration
type MetricsConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Path      string `mapstructure:"path"`
	Port      int    `mapstructure:"port"`
	Namespace string `mapstructure:"namespace"`
}

// TracingConfig represents tracing configuration
type TracingConfig struct {
	Enabled     bool    `mapstructure:"enabled"`
	ServiceName string  `mapstructure:"service_name"`
	Endpoint    string  `mapstructure:"endpoint"`
	SampleRate  float64 `mapstructure:"sample_rate"`
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	Enabled bool `mapstructure:"enabled"`
	Global  struct {
		RPS   int           `mapstructure:"rps"`
		Burst int           `mapstructure:"burst"`
		TTL   time.Duration `mapstructure:"ttl"`
	} `mapstructure:"global"`
	PerUser struct {
		RPS   int           `mapstructure:"rps"`
		Burst int           `mapstructure:"burst"`
		TTL   time.Duration `mapstructure:"ttl"`
	} `mapstructure:"per_user"`
	PerIP struct {
		RPS   int           `mapstructure:"rps"`
		Burst int           `mapstructure:"burst"`
		TTL   time.Duration `mapstructure:"ttl"`
	} `mapstructure:"per_ip"`
}

// CircuitBreakConfig represents circuit breaker configuration
type CircuitBreakConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	MaxRequests     uint32        `mapstructure:"max_requests"`
	Interval        time.Duration `mapstructure:"interval"`
	Timeout         time.Duration `mapstructure:"timeout"`
	FailureRatio    float64       `mapstructure:"failure_ratio"`
	MinRequestCount uint32        `mapstructure:"min_request_count"`
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	Local struct {
		Enabled  bool          `mapstructure:"enabled"`
		Size     int           `mapstructure:"size"`     
		TTL      time.Duration `mapstructure:"ttl"`      
		CleanupInterval time.Duration `mapstructure:"cleanup_interval"`
	} `mapstructure:"local"`
	Redis struct {
		Enabled    bool          `mapstructure:"enabled"`
		KeyPrefix  string        `mapstructure:"key_prefix"`
		DefaultTTL time.Duration `mapstructure:"default_ttl"`
	} `mapstructure:"redis"`
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	JWT struct {
		Secret     string        `mapstructure:"secret"`
		Expire     time.Duration `mapstructure:"expire"`
		RefreshTTL time.Duration `mapstructure:"refresh_ttl"`
		Issuer     string        `mapstructure:"issuer"`
	} `mapstructure:"jwt"`
	CORS struct {
		Enabled          bool     `mapstructure:"enabled"`
		AllowOrigins     []string `mapstructure:"allow_origins"`
		AllowMethods     []string `mapstructure:"allow_methods"`
		AllowHeaders     []string `mapstructure:"allow_headers"`
		ExposeHeaders    []string `mapstructure:"expose_headers"`
		AllowCredentials bool     `mapstructure:"allow_credentials"`
		MaxAge           int      `mapstructure:"max_age"`
	} `mapstructure:"cors"`
	Encryption struct {
		Key string `mapstructure:"key"`
		IV  string `mapstructure:"iv"`
	} `mapstructure:"encryption"`
}

// SeckillConfig represents seckill business configuration
type SeckillConfig struct {
	StockCache struct {
		Enabled    bool          `mapstructure:"enabled"`
		TTL        time.Duration `mapstructure:"ttl"`
		KeyPrefix  string        `mapstructure:"key_prefix"`
		PreloadNum int           `mapstructure:"preload_num"` 
	} `mapstructure:"stock_cache"`
	Order struct {
		Timeout       time.Duration `mapstructure:"timeout"`       
		RetryTimes    int           `mapstructure:"retry_times"`    
		RetryInterval time.Duration `mapstructure:"retry_interval"` 
	} `mapstructure:"order"`
	Activity struct {
		PreloadTime time.Duration `mapstructure:"preload_time"` 
		CacheTime   time.Duration `mapstructure:"cache_time"`  
	} `mapstructure:"activity"`
}

// GetAddr returns the server address
func (s *ServerConfig) GetAddr() string {
	if s.Host == "" {
		s.Host = "0.0.0.0"
	}
	if s.Port == 0 {
		s.Port = 8080
	}
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// GetDSN returns the database DSN
func (d *DatabaseConfig) GetDSN() string {
	if d.Charset == "" {
		d.Charset = "utf8mb4"
	}
	if d.Loc == "" {
		d.Loc = "Local"
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		d.Username, d.Password, d.Host, d.Port, d.DBName, d.Charset, d.ParseTime, d.Loc)
}

// GetAddr returns the Redis address
func (r *RedisConfig) GetAddr() string {
	if r.Host == "" {
		r.Host = "localhost"
	}
	if r.Port == 0 {
		r.Port = 6379
	}
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	
	if c.Database.Username == "" {
		return fmt.Errorf("database username is required")
	}
	
	if c.Database.DBName == "" {
		return fmt.Errorf("database name is required")
	}
	
	if c.Redis.Host == "" {
		return fmt.Errorf("redis host is required")
	}
	
	if c.Security.JWT.Secret == "" {
		return fmt.Errorf("JWT secret is required")
	}
	
	return nil
}

// SetDefaults sets default values for configuration
func (c *Config) SetDefaults() {
	if c.Server.Host == "" {
		c.Server.Host = "localhost"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.Mode == "" {
		c.Server.Mode = "debug"
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 30 * time.Second
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 30 * time.Second
	}
	if c.Server.IdleTimeout == 0 {
		c.Server.IdleTimeout = 60 * time.Second
	}
	if c.Server.MaxHeaderMB == 0 {
		c.Server.MaxHeaderMB = 1
	}

	if c.Database.Driver == "" {
		c.Database.Driver = "mysql"
	}
	if c.Database.Host == "" {
		c.Database.Host = "localhost"
	}
	if c.Database.Port == 0 {
		c.Database.Port = 3306
	}
	if c.Database.Charset == "" {
		c.Database.Charset = "utf8mb4"
	}
	if c.Database.Loc == "" {
		c.Database.Loc = "Local"
	}
	if c.Database.MaxOpenConns == 0 {
		c.Database.MaxOpenConns = 100
	}
	if c.Database.MaxIdleConns == 0 {
		c.Database.MaxIdleConns = 10
	}
	if c.Database.ConnMaxLifetime == 0 {
		c.Database.ConnMaxLifetime = time.Hour
	}
	if c.Database.ConnMaxIdleTime == 0 {
		c.Database.ConnMaxIdleTime = 10 * time.Minute
	}
	if c.Database.LogLevel == "" {
		c.Database.LogLevel = "warn"
	}

	if c.Redis.Host == "" {
		c.Redis.Host = "localhost"
	}
	if c.Redis.Port == 0 {
		c.Redis.Port = 6379
	}
	if c.Redis.PoolSize == 0 {
		c.Redis.PoolSize = 100
	}
	if c.Redis.MinIdleConns == 0 {
		c.Redis.MinIdleConns = 10
	}
	if c.Redis.MaxRetries == 0 {
		c.Redis.MaxRetries = 3
	}
	if c.Redis.DialTimeout == 0 {
		c.Redis.DialTimeout = 5 * time.Second
	}
	if c.Redis.ReadTimeout == 0 {
		c.Redis.ReadTimeout = 3 * time.Second
	}
	if c.Redis.WriteTimeout == 0 {
		c.Redis.WriteTimeout = 3 * time.Second
	}
	if c.Redis.PoolTimeout == 0 {
		c.Redis.PoolTimeout = 4 * time.Second
	}
	if c.Redis.IdleTimeout == 0 {
		c.Redis.IdleTimeout = 5 * time.Minute
	}

	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
	if c.Log.Format == "" {
		c.Log.Format = "json"
	}
	if c.Log.Output == "" {
		c.Log.Output = "stdout"
	}

	if c.Security.JWT.Expire == 0 {
		c.Security.JWT.Expire = 2 * time.Hour
	}
	if c.Security.JWT.RefreshTTL == 0 {
		c.Security.JWT.RefreshTTL = 7 * 24 * time.Hour
	}
	if c.Security.JWT.Issuer == "" {
		c.Security.JWT.Issuer = "seckill-system"
	}

	if c.Seckill.StockCache.TTL == 0 {
		c.Seckill.StockCache.TTL = 10 * time.Minute
	}
	if c.Seckill.StockCache.KeyPrefix == "" {
		c.Seckill.StockCache.KeyPrefix = "stock:"
	}
	if c.Seckill.Order.Timeout == 0 {
		c.Seckill.Order.Timeout = 15 * time.Minute
	}
	if c.Seckill.Order.RetryTimes == 0 {
		c.Seckill.Order.RetryTimes = 3
	}
	if c.Seckill.Order.RetryInterval == 0 {
		c.Seckill.Order.RetryInterval = time.Second
	}
	if c.Seckill.Activity.PreloadTime == 0 {
		c.Seckill.Activity.PreloadTime = 10 * time.Minute
	}
	if c.Seckill.Activity.CacheTime == 0 {
		c.Seckill.Activity.CacheTime = 5 * time.Minute
	}
}