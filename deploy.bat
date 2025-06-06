@echo off
REM ========================
REM Script Otomatis Build, Upload, dan Jalankan Go di VPS Niagahoster
REM ========================

REM 0. Pastikan semua file disimpan sebelum menjalankan script ini!
echo PASTIKAN SEMUA PERUBAHAN FILE SUDAH DISIMPAN!
pause

REM 1. Build aplikasi Go untuk Linux dari cmd/main.go
echo Building Go application for Linux...
set CGO_ENABLED=0
set GOOS=linux
set GOARCH=amd64
go build -a -o app ./cmd/app
IF %ERRORLEVEL% NEQ 0 (
    echo ERROR: Go build failed!
    pause
    exit /b %ERRORLEVEL%
)
echo Build successful.

REM 2. Hapus file lama di VPS (untuk kebersihan)
echo Deleting old files on VPS...
"C:\Program Files\PuTTY\plink.exe" -batch -ssh root@31.97.48.130 -pw Qwewenzqwewenz1@ "rm -f /root/tmp/app /root/app"

REM 2.A HAPUS FOLDER CONFIG LAMA DI VPS (BARU)
echo Deleting old config folder on VPS...
"C:\Program Files\PuTTY\plink.exe" -batch -ssh root@31.97.48.130 -pw Qwewenzqwewenz1@ "rm -f /root/config.yaml"

REM 3. Pastikan folder /root/tmp ada di VPS
echo Creating temp folder on VPS...
"C:\Program Files\PuTTY\plink.exe" -batch -ssh root@31.97.48.130 -pw Qwewenzqwewenz1@ "mkdir -p /root/tmp"

REM 4. Upload binary app ke VPS (ke folder tmp)
echo Uploading application binary to VPS...
"C:\Program Files\PuTTY\pscp.exe" -pw Qwewenzqwewenz1@ app root@31.97.48.130:/root/tmp/app
IF %ERRORLEVEL% NEQ 0 (
    echo ERROR: Uploading application binary failed!
    pause
    exit /b %ERRORLEVEL%
)

REM 5. Upload folder web ke VPS
echo Uploading web folder to VPS...
"C:\Program Files\PuTTY\pscp.exe" -r -pw Qwewenzqwewenz1@ web root@31.97.48.130:/root/

REM 6. Upload folder config ke VPS
echo Uploading config folder to VPS...
"C:\Program Files\PuTTY\pscp.exe" -pw Qwewenzqwewenz1@ config.yaml root@31.97.48.130:/root/config.yaml

REM 6.A. Buat direktori untuk migrasi di VPS
echo Creating migrations directory on VPS...
"C:\Program Files\PuTTY\plink.exe" -batch -ssh root@31.97.48.130 -pw Qwewenzqwewenz1@ "mkdir -p /root/internal/database/migrations"

REM 6.B. Upload migration files ke VPS (satu per satu untuk kejelasan)
echo Uploading migration files to VPS...
"C:\Program Files\PuTTY\pscp.exe" -pw Qwewenzqwewenz1@ internal\database\migrations\000001_initial_schema.up.sql root@31.97.48.130:/root/internal/database/migrations/
IF %ERRORLEVEL% NEQ 0 (
    echo ERROR: Uploading 000001_initial_schema.up.sql failed!
    pause
    exit /b %ERRORLEVEL%
)
"C:\Program Files\PuTTY\pscp.exe" -pw Qwewenzqwewenz1@ internal\database\migrations\000001_initial_schema.down.sql root@31.97.48.130:/root/internal/database/migrations/
IF %ERRORLEVEL% NEQ 0 (
    echo ERROR: Uploading 000001_initial_schema.down.sql failed!
    pause
    exit /b %ERRORLEVEL%
)

REM 7. Jalankan skrip aksi jarak jauh di VPS
echo Executing remote actions script on VPS via plink -m...
"C:\Program Files\PuTTY\plink.exe" -batch -ssh root@31.97.48.130 -pw Qwewenzqwewenz1@ -m remote_actions.sh

REM 11. Tunggu sebentar (opsional, karena skrip remote sudah menunggu)
REM timeout /t 3 > NUL 

REM 12. Tampilkan status aplikasi (opsional, karena skrip remote sudah menampilkan status dan log)
REM echo Checking if application is running...
REM "C:\Program Files\PuTTY\plink.exe" -batch -ssh root@31.97.48.130 -pw Qwewenzqwewenz1@ "ps aux | grep ./app | grep -v grep"

echo ==========================
echo Selesai! Aplikasi sudah dijalankan di VPS.
echo Cek di browser: http://31.97.48.130:8080/
echo Untuk melihat log: ssh root@31.97.48.130 "tail -f /root/app.log"
echo ==========================

REM 13. Buka jendela CMD baru dan jalankan tail -f untuk log aplikasi
echo Membuka log aplikasi di jendela baru (tail -f)...
start "VPS App Log" cmd /k ""C:\\Program Files\\PuTTY\\plink.exe" -ssh root@31.97.48.130 -pw Qwewenzqwewenz1@ "tail -f /root/app.log""

pause