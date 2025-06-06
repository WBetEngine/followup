#!/bin/bash
echo "--- Memulai skrip deployment jarak jauh (vSmartMigrate) ---"

# Langkah 0: Pindah ke direktori /root sebagai CWD
echo "0. Pindah ke direktori /root"
cd /root || { echo "FATAL: Gagal pindah ke /root"; exit 1; }

# Langkah 1: Hentikan aplikasi lama (jika berjalan)
echo "1. Menghentikan aplikasi lama yang mungkin berjalan di port 8080"
fuser -k -n tcp 8080 || echo "Port 8080 sudah bebas atau fuser gagal (diabaikan)."
sleep 2 # Beri waktu proses untuk berhenti

# Langkah 2: Pindahkan biner baru dan atur izin
echo "2. Memindahkan biner aplikasi baru dan mengatur izin"
mv -f /root/tmp/app /root/app || { echo "FATAL: Gagal memindahkan biner aplikasi"; exit 1; }
chmod +x /root/app || { echo "FATAL: Gagal mengatur izin eksekusi"; exit 1; }

# Langkah 3: Mengatur variabel environment
echo "3. Mengatur variabel environment"
export DATABASE_URL='postgres://Grup403:Qwewenz1*@localhost:5432/Teledb?sslmode=disable'
export SESSION_KEY='g_t_n_V8e_q_S_l_4_p_X_w_A_j_u_O_0_k_R_y_I_b_F_z_m_D_h_C_2_f_E_1_Y_7_U_i_N_5_T_J_c_B_3_P_K_d_L_9_Q_6'
export JWT_SECRET='s3cr3t_K3y_F0r_JWT_@uth_Th!$_!s_L0ng_&_Secure_g_t_n_V8e_q_S_l_4_p_X_w_A_j_u_O_0_k_R_y_I_b_F_z_m_D_h_C_2_f_E_1_Y_7_U_i_N_5_T_J_c_B_3_P_K_d_L_9_Q_6_!@#$%^&*()'

# Langkah 4: Jalankan Migrasi Otomatis dengan Penanganan Error
echo "4. Menjalankan migrasi database otomatis..."
# Jalankan migrasi 'up' dan simpan outputnya (termasuk stderr) ke variabel
MIGRATE_LOG=$(./app -migrate-up 2>&1)
MIGRATE_EXIT_CODE=$?

echo "--- Log Migrasi Awal ---"
echo "$MIGRATE_LOG"
echo "--- Selesai Log Migrasi Awal ---"

# Periksa apakah migrasi awal gagal DAN apakah log mengandung kata "dirty"
if [ $MIGRATE_EXIT_CODE -ne 0 ] && echo "$MIGRATE_LOG" | grep -q "dirty"; then
    echo "   -> Migrasi awal gagal dan database terdeteksi 'dirty'."
    echo "   -> Mencoba perbaikan otomatis dengan '-migrate-force=1'..."
    
    FORCE_LOG=$(./app -migrate-force=1 2>&1)
    echo "--- Log Force Migration ---"
    echo "$FORCE_LOG"
    echo "--- Selesai Log Force Migration ---"

    if [ $? -eq 0 ]; then
        echo "   -> Force migration berhasil. Mencoba 'migrate-up' sekali lagi..."
        RETRY_MIGRATE_LOG=$(./app -migrate-up 2>&1)
        echo "--- Log Migrasi Ulang ---"
        echo "$RETRY_MIGRATE_LOG"
        echo "--- Selesai Log Migrasi Ulang ---"
        if [ $? -ne 0 ]; then
             echo "FATAL: Migrasi tetap gagal bahkan setelah perbaikan otomatis. Silakan periksa log di atas."
             exit 1
        fi
        echo "   -> Perbaikan otomatis dan migrasi ulang berhasil."
    else
        echo "FATAL: Gagal menjalankan force migration. Silakan periksa log di atas."
        exit 1
    fi
elif [ $MIGRATE_EXIT_CODE -ne 0 ]; then
    echo "FATAL: Migrasi gagal karena alasan selain 'dirty state'. Silakan periksa log migrasi awal di atas."
    exit 1
else
    echo "   -> Migrasi berhasil pada percobaan pertama."
fi


# Langkah 5: Jalankan Aplikasi Utama
echo "5. Menjalankan aplikasi utama..."
nohup ./app > app.log 2>&1 &
sleep 3 # Beri waktu aplikasi untuk mulai

# Langkah 6: Verifikasi Status Aplikasi
echo "6. Memeriksa status aplikasi..."
if pgrep -f "/root/app" > /dev/null; then
    echo "   -> SUKSES: Aplikasi '/root/app' terdeteksi berjalan."
    echo "   -> Detail proses:"
    ps aux | grep '[/]root/app'
else
    echo "   -> GAGAL: Aplikasi '/root/app' tidak terdeteksi berjalan setelah startup."
fi

# Langkah 7: Tampilkan Log Terakhir
echo "7. Menampilkan 15 baris terakhir dari log aplikasi (app.log)"
tail -n 15 app.log

echo "--- Skrip deployment jarak jauh (vSmartMigrate) selesai ---"