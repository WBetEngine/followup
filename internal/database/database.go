package database

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"followup/internal/config"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	// Tambahkan impor untuk golang-migrate
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres" // Driver PostgreSQL untuk migrate
	_ "github.com/golang-migrate/migrate/v4/source/file"     // Driver untuk membaca migrasi dari file
)

// DB adalah instansi koneksi database global
var (
	db   *sql.DB
	once sync.Once
)

// GetDB mengembalikan koneksi database global yang sudah diinisialisasi.
// Fungsi ini memastikan bahwa koneksi database hanya diinisialisasi sekali.
func GetDB() *sql.DB {
	once.Do(func() {
		var err error
		cfg := config.GetConfig()

		// Buat koneksi database berdasarkan driver yang dikonfigurasi.
		db, err = sql.Open(cfg.Database.Driver, cfg.Database.DSN())
		if err != nil {
			log.Fatalf("Gagal membuka koneksi ke database: %v", err)
		}

		// Konfigurasi properti koneksi pool untuk optimalisasi performa.
		db.SetMaxOpenConns(cfg.Database.MaxOpenConns)                                           // Ambil dari config
		db.SetMaxIdleConns(cfg.Database.MaxIdleConns)                                           // Ambil dari config
		db.SetConnMaxLifetime(time.Duration(cfg.Database.ConnMaxLifetimeMinutes) * time.Minute) // Ambil dari config

		// Cek koneksi ke database untuk memastikan konfigurasi sudah benar.
		if err = db.Ping(); err != nil {
			log.Fatalf("Gagal melakukan ping ke database: %v", err)
		}

		log.Printf("Berhasil terhubung ke database %s", cfg.Database.Driver)
	})

	return db
}

// getMigrateInstance adalah helper internal untuk membuat instance migrate.
func getMigrateInstance() (*migrate.Migrate, error) {
	conn := GetDB()
	cfg := config.GetConfig()
	if cfg.Database.Driver != "postgres" {
		return nil, fmt.Errorf("skipping migrations: driver is not postgres")
	}

	driver, err := postgres.WithInstance(conn, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("could not create postgres driver instance: %w", err)
	}

	// Asumsi file migrasi ada di <cwd>/internal/database/migrations
	// Ini akan bekerja baik di lokal maupun di server di mana CWD adalah /root
	migrationsPath := "internal/database/migrations"

	// Gunakan path absolut untuk kejelasan dalam log
	absMigrationsPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		log.Printf("Warning: could not get absolute path for migrations: %v", err)
		absMigrationsPath = migrationsPath // fallback ke path relatif
	}

	migrationsSourceURL := "file://" + absMigrationsPath
	log.Printf("DEBUG: Using migration source URL: %s", migrationsSourceURL)

	m, err := migrate.NewWithDatabaseInstance(migrationsSourceURL, "postgres", driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance (path: %s): %w", absMigrationsPath, err)
	}
	return m, nil
}

// RunMigrations menjalankan migrasi 'up'. Tidak akan crash jika DB dirty.
func RunMigrations() {
	log.Println("Checking database migrations...")
	m, err := getMigrateInstance()
	if err != nil {
		if err.Error() == "skipping migrations: driver is not postgres" {
			log.Println(err)
		} else {
			log.Printf("Could not get migrate instance: %v. Skipping migrations.", err)
		}
		return
	}

	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			log.Println("Database is already up to date.")
		} else {
			log.Printf("An error occurred during 'up' migration: %v", err)
			// Periksa apakah DB dalam keadaan kotor setelah error
			_, dirty, _ := m.Version()
			if dirty {
				log.Println("WARNING: Database is in a dirty state. The application might not function correctly. Run the app with -migrate-force=<version> to fix.")
			}
		}
	} else {
		log.Println("Database migrations applied successfully.")
	}

	version, dirty, _ := m.Version()
	log.Printf("Current DB version: %d, Dirty: %t", version, dirty)
}

// MigrateDown menjalankan satu migrasi 'down'.
func MigrateDown() error {
	m, err := getMigrateInstance()
	if err != nil {
		return err
	}
	log.Println("Running one 'down' migration...")
	if err := m.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("an error occurred during 'down' migration: %w", err)
	}
	log.Println("Down migration applied successfully.")
	return nil
}

// ForceMigrationVersion memaksa database ke versi tertentu, membersihkan status 'dirty'.
func ForceMigrationVersion(version int) error {
	m, err := getMigrateInstance()
	if err != nil {
		return err
	}
	log.Printf("Forcing migration version to %d...", version)
	if err := m.Force(version); err != nil {
		return fmt.Errorf("failed to force version %d: %w", version, err)
	}
	log.Printf("Successfully forced migration version to %d.", version)
	return nil
}
