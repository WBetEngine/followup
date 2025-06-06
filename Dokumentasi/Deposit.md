# Dokumentasi Fitur Deposit

## Pendahuluan
Fitur deposit adalah bagian dari sistem yang menangani proses deposit member, mulai dari input hingga persetujuan.

## Fitur Utama

1. **Input Deposit**
   - Modal popup untuk input deposit
   - Field input:
     - Jumlah Deposit (format currency Rp)
     - Nama (auto-filled dari data member)
     - Role (auto-filled dari user yang login)
     - Option Brand yang diambil dari database Brand
   - Tombol:
     - Submit (primary)
     - Cancel (secondary)

2. **Proses Persetujuan**
   - Hanya Admin dan Superadmin yang dapat menyetujui deposit
   - Tidak ada batasan waktu untuk pembatalan deposit
   - Deposit yang sudah approved bisa dibatalkan
   - Notifikasi ke CRM saat deposit dibatalkan dan disetujui

3. **History Approve Deposit**
   - Menampilkan riwayat semua transaksi deposit
   - Pagination: 50 data per halaman
   - Informasi yang ditampilkan:
     - Tanggal dan waktu transaksi
     - Username member
     - Jumlah deposit
     - Status (Pending/Approved/Rejected)
     - Role yang melakukan approve/reject
     - Brand yang dipilih
   - Fitur filter berdasarkan:
     - Tanggal
     - Status
     - Username (dengan fitur pencarian)
     - Brand

## Batasan dan Validasi

1. **Deposit**
   - Maksimum deposit per transaksi: Rp 10.000
   - Logging untuk setiap akses ke halaman deposit

2. **Pembatalan Deposit**
   - Hanya admin/superadmin yang dapat membatalkan
   - Deposit yang sudah approved bisa dibatalkan
   - Notifikasi ke CRM saat deposit dibatalkan dan disetujui
   - Logging untuk setiap pembatalan dan persetujuan 