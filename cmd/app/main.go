package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"followup/internal/config"
	"followup/internal/database"
	"followup/internal/handlers"
	"followup/internal/render"
	"followup/internal/repository"
	"followup/internal/router"
	"followup/internal/services"
)

func main() {
	// --- MODE MIGRASI ---
	// Menambahkan flag untuk mode migrasi
	migrateUp := flag.Bool("migrate-up", false, "Run 'up' migrations and exit")
	migrateDown := flag.Bool("migrate-down", false, "Run 'down' migrations and exit")
	forceVersion := flag.Int("migrate-force", -1, "Force migration version and exit")
	flag.Parse()

	// Jika salah satu flag migrasi digunakan, jalankan aksi dan keluar.
	// Kita perlu init config dan db dulu agar migrasi bisa berjalan.
	if *migrateUp || *migrateDown || *forceVersion != -1 {
		log.Println("--- RUNNING IN MIGRATION MODE ---")
		_ = config.GetConfig() // Pastikan config dimuat
		_ = database.GetDB()   // Pastikan koneksi DB dibuat

		if *migrateUp {
			database.RunMigrations() // Fungsi ini sudah mencakup logging
		}
		if *migrateDown {
			if err := database.MigrateDown(); err != nil {
				log.Fatalf("FATAL: Down migration failed: %v", err)
			}
		}
		if *forceVersion != -1 {
			if err := database.ForceMigrationVersion(*forceVersion); err != nil {
				log.Fatalf("FATAL: Failed to force migration version: %v", err)
			}
		}
		log.Println("--- MIGRATION MODE FINISHED ---")
		os.Exit(0) // Keluar setelah mode migrasi selesai
	}

	// --- LOGIKA APLIKASI NORMAL (Tidak berubah dari kode asli Anda) ---
	log.Println("--- RUNNING IN NORMAL SERVER MODE ---")

	// Initialize template cache
	if err := render.InitTemplates(); err != nil {
		log.Fatalf("Error initializing templates: %v", err)
	}

	// Load config
	cfg := config.GetConfig()

	// Initialize database connection
	dbInstance := database.GetDB()
	if dbInstance == nil {
		log.Fatalf("Failed to get database instance")
	}

	// Jalankan migrasi database (sebagai pemeriksaan rutin)
	database.RunMigrations()

	// Initialize repositories
	brandRepo := repository.NewBrandRepository(dbInstance)
	memberRepo := repository.NewMemberRepository(dbInstance)
	userRepo := repository.NewUserRepository(dbInstance)

	// Initialize services
	brandService := services.NewBrandService(brandRepo)
	memberService := services.NewMemberService(memberRepo, userRepo)

	// Initialize handlers (struct-based handlers)
	brandStructHandler := handlers.NewBrandHandler(brandService)

	// Set up router
	r := router.SetupRouter(dbInstance, brandService, brandStructHandler, memberService)

	// Create server
	addr := fmt.Sprintf("%s%s", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
}
