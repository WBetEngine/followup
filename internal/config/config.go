package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config struktur untuk menyimpan konfigurasi aplikasi
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Auth     AuthConfig     `yaml:"auth"`
	App      AppConfig      `yaml:"app"`
	Session  SessionConfig  `yaml:"session"`
}

// ServerConfig menyimpan konfigurasi server
type ServerConfig struct {
	Port         string `yaml:"port"`
	Host         string `yaml:"host"`
	ReadTimeout  int    `yaml:"read_timeout"`
	WriteTimeout int    `yaml:"write_timeout"`
	IdleTimeout  int    `yaml:"idle_timeout"`
}

// DatabaseConfig menyimpan konfigurasi database
type DatabaseConfig struct {
	Driver                 string `yaml:"driver"`
	URL                    string `yaml:"url"`      // URL koneksi langsung (prioritas)
	Host                   string `yaml:"host"`     // Digunakan jika URL kosong
	Port                   string `yaml:"port"`     // Digunakan jika URL kosong
	User                   string `yaml:"user"`     // Digunakan jika URL kosong
	Password               string `yaml:"password"` // Digunakan jika URL kosong
	DBName                 string `yaml:"dbname"`   // Digunakan jika URL kosong
	SSLMode                string `yaml:"sslmode"`  // Digunakan jika URL kosong
	MaxOpenConns           int    `yaml:"max_open_conns"`
	MaxIdleConns           int    `yaml:"max_idle_conns"`
	ConnMaxLifetimeMinutes int    `yaml:"conn_max_lifetime_minutes"`
}

// AuthConfig menyimpan konfigurasi otentikasi
type AuthConfig struct {
	JWTSecret     string `yaml:"jwt_secret"`
	JWTExpiration int    `yaml:"jwt_expiration"` // Dalam jam
}

// AppConfig menyimpan konfigurasi aplikasi
type AppConfig struct {
	Name           string `yaml:"name"`
	Environment    string `yaml:"environment"`
	LogLevel       string `yaml:"log_level"`
	StaticDir      string `yaml:"static_dir"`
	TemplatesDir   string `yaml:"templates_dir"`
	MaxUploadSize  int64  `yaml:"max_upload_size"`
	AllowedDomains string `yaml:"allowed_domains"`
}

// SessionConfig menyimpan konfigurasi session
type SessionConfig struct {
	Key    string `yaml:"key"`
	MaxAge int    `yaml:"max_age"` // Dalam detik
}

var (
	cfg  *Config
	once sync.Once
)

// DSN mengembalikan string koneksi untuk database
func (c *DatabaseConfig) DSN() string {
	// Jika URL disediakan, gunakan itu
	if c.URL != "" {
		return c.URL
	}

	// Jika tidak, bangun DSN dari komponen individu
	switch c.Driver {
	case "postgres":
		return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
			c.User, c.Password, c.Host, c.Port, c.DBName)
	case "sqlite3":
		return c.DBName
	default:
		return ""
	}
}

// GetConfig mengembalikan instance konfigurasi (singleton)
func GetConfig() *Config {
	once.Do(func() {
		cfg = &Config{}
		if err := loadConfig(cfg); err != nil {
			log.Fatalf("Error loading config: %v", err)
		}
	})
	return cfg
}

// loadConfig memuat konfigurasi dari file config.yaml
func loadConfig(cfg *Config) error {
	// Mencari file konfigurasi
	configFile := filepath.Join(".", "config.yaml")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// Coba cari di direktori parent
		configFile = filepath.Join("..", "config.yaml")
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			return fmt.Errorf("config file not found: %v", err)
		}
	}

	// Membaca file konfigurasi
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("error reading config file: %v", err)
	}

	// Parsing YAML
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("error parsing config file: %v", err)
	}

	// Menggunakan environment variables jika ada
	setFromEnv(cfg)

	return nil
}

// setFromEnv memperbarui nilai konfigurasi dari environment variables jika ada
func setFromEnv(cfg *Config) {
	// Server
	if port := os.Getenv("SERVER_PORT"); port != "" {
		cfg.Server.Port = port
	}

	// Database
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		cfg.Database.URL = dbURL
	}
	if dbUser := os.Getenv("DB_USER"); dbUser != "" {
		cfg.Database.User = dbUser
	}
	if dbPass := os.Getenv("DB_PASSWORD"); dbPass != "" {
		cfg.Database.Password = dbPass
	}
	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		cfg.Database.Host = dbHost
	}
	if dbPort := os.Getenv("DB_PORT"); dbPort != "" {
		cfg.Database.Port = dbPort
	}
	if dbName := os.Getenv("DB_NAME"); dbName != "" {
		cfg.Database.DBName = dbName
	}

	// Auth
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		cfg.Auth.JWTSecret = jwtSecret
	}

	// Session
	if sessionKey := os.Getenv("SESSION_KEY"); sessionKey != "" {
		cfg.Session.Key = sessionKey
	}

	// App
	if env := os.Getenv("APP_ENV"); env != "" {
		cfg.App.Environment = env
	}
}
