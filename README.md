

```
followup/
├── cmd/
│   └── app/
│       └── main.go          # Entry point aplikasi
├── internal/
│   ├── auth/                # Autentikasi
│   ├── config/              # Konfigurasi
│   ├── database/            # Database
│   │   └── migrations/      # Migrasi database
│   ├── handlers/            # HTTP handlers
│   ├── middleware/          # Middleware
│   ├── models/              # Model data
│   ├── render/              # Template rendering
│   └── router/              # Routing
├── web/
│   ├── static/              # Aset statis (CSS, JS, gambar)
│   └── templates/           # Template HTML
│       ├── pages/           # Halaman
│       └── partials/        # Komponen yang dapat digunakan kembali
├── config.yaml              # File konfigurasi
├── go.mod                   # Dependensi Go
└── README.md                # Dokumentasi
```
