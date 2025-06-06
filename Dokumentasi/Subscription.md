# Dokumentasi Halaman Langganan

## Pendahuluan
Halaman Langganan adalah halaman yang menampilkan data member yang sudah melakukan deposit dan telah disetujui di halaman Deposit. Halaman ini terintegrasi dengan berbagai halaman lain seperti Deposit, Withdrawal, dan Followup.

## Fitur Utama

1. **Tampilan Data Langganan**
   - Data member yang sudah deposit dan disetujui
   - Informasi yang ditampilkan:
     - Username
     - Email
     - Nomor Telepon
     - Bank
     - Nomor Rekening
     - Brand
     - Status
     - CRM yang menangani
     - Tanggal deposit
     - Jumlah deposit
     - Team
   - Tombol aksi di setiap baris:
     - Tombol Deposit (untuk input deposit baru)
     - Tombol History Deposit (untuk melihat riwayat deposit)

2. **Filter dan Pencarian**
   - Filter berdasarkan:
     - Tanggal deposit
     - Status
     - Brand
     - Team
     - CRM
   - Pencarian berdasarkan:
     - Username
     - Email
     - Nomor Telepon
     - Nomor Rekening

3. **Akses Berdasarkan Role**
   - Superadmin:
     - Dapat melihat semua data langganan
     - Akses ke semua team dan CRM
     - Dapat melakukan semua operasi
   - Admin:
     - Hanya dapat melihat data team yang dikelola
     - Dapat melihat data CRM dalam team
     - Dapat melakukan operasi terbatas
   - CRM:
     - Hanya dapat melihat data langganan sendiri
     - Tidak dapat melihat data CRM lain
     - Operasi terbatas pada data sendiri

4. **Fitur Deposit**
   - Modal popup untuk input deposit baru
   - Field input:
     - Jumlah Deposit (format currency Rp)
     - Nama (auto-filled dari data member)
     - Role (auto-filled dari user yang login)
     - Option Brand yang diambil dari database Brand
   - Tombol:
     - Submit (primary)
     - Cancel (secondary)
   - Validasi input
   - Notifikasi setelah submit

5. **Fitur History Deposit**
   - Modal popup menampilkan riwayat deposit member
   - Informasi yang ditampilkan:
     - Tanggal deposit
     - Jumlah deposit
     - Status (Approved/Rejected)
     - Role yang melakukan input
     - Role yang melakukan approve/reject
     - Brand yang dipilih
   - Filter berdasarkan:
     - Tanggal
     - Status
     - Brand
   - Pencarian berdasarkan:
     - Tanggal
     - Jumlah
     - Status

6. **Sistem Rotasi Member**
   - Pengecekan otomatis setiap hari untuk member yang tidak melakukan deposit
   - Kriteria rotasi:
     - Tidak ada deposit dalam 2 minggu
     - Tidak ada input deposit yang disetujui
   - Proses rotasi:
     - Data member dipindahkan ke halaman Followup
     - CRM ditentukan secara otomatis menggunakan sistem round-robin
     - Notifikasi dikirim ke CRM baru
     - Status member diupdate di database
   - Informasi yang dirotasi:
     - Data member lengkap
     - Riwayat deposit sebelumnya
     - Status terakhir
   - Logging untuk setiap proses rotasi

## Integrasi

1. **Integrasi dengan Halaman Deposit**
   - Data otomatis masuk setelah deposit disetujui
   - Status deposit tercermin di halaman langganan
   - Riwayat deposit dapat diakses

2. **Integrasi dengan Halaman Withdrawal**
   - Data withdrawal member tercatat
   - Status withdrawal tercermin
   - Riwayat withdrawal dapat diakses

3. **Integrasi dengan Halaman Followup**
   - Data followup tercatat
   - Status followup tercermin
   - Riwayat followup dapat diakses

4. **Integrasi dengan Sistem Team**
   - Data team tercermin di halaman langganan
   - Pembatasan akses berdasarkan team
   - Statistik per team

5. **Integrasi dengan Sistem Rotasi**
   - Pengecekan otomatis setiap hari
   - Integrasi dengan halaman Followup
   - Sistem round-robin untuk penentuan CRM
   - Notifikasi otomatis ke CRM baru
   - Update status di database members

## Alur Kerja

1. **Proses Deposit ke Langganan**
   - Member melakukan deposit
   - Deposit disetujui di halaman Deposit
   - Data otomatis masuk ke halaman Langganan
   - Status diperbarui

2. **Proses Input Deposit Baru**
   - User mengklik tombol Deposit
   - Modal popup muncul
   - User mengisi form deposit
   - Sistem memvalidasi input
   - Data dikirim ke halaman Deposit
   - Notifikasi dikirim ke pihak terkait

3. **Proses Melihat History Deposit**
   - User mengklik tombol History Deposit
   - Modal popup menampilkan riwayat
   - User dapat memfilter dan mencari data
   - Data dapat di-export (untuk admin/superadmin)

4. **Proses Akses Data**
   - User login dengan role tertentu
   - Sistem mengecek akses dan team
   - Data ditampilkan sesuai akses
   - Filter dan pencarian tersedia

5. **Proses Update Data**
   - Data diupdate dari halaman terkait
   - Status diperbarui otomatis
   - Riwayat perubahan tercatat
   - Notifikasi ke pihak terkait

6. **Proses Rotasi Member**
   - Sistem mengecek status deposit setiap hari
   - Jika tidak ada deposit dalam 2 minggu:
     - Data dipindahkan ke halaman Followup dengan status Redeposit
     - CRM baru ditentukan secara otomatis
     - Notifikasi dikirim ke CRM baru
     - Status diperbarui di database
   - CRM baru dapat:
     - Melakukan followup
     - Input deposit baru
     - Melihat riwayat deposit sebelumnya

## Spesifikasi Teknis

1. **Sistem Rotasi**
   - Pengecekan harian menggunakan cron job
   - Perhitungan 2 minggu berdasarkan tanggal deposit terakhir
   - Sistem round-robin untuk distribusi member ke CRM
   - Logging untuk setiap proses rotasi
   - Notifikasi real-time ke CRM baru

## Batasan dan Validasi

4. **Sistem Rotasi**
   - Batas waktu: 2 minggu tanpa deposit
   - Pengecekan: Setiap hari pada waktu yang ditentukan
   - Notifikasi wajib ke CRM baru
   - Logging wajib untuk setiap rotasi 