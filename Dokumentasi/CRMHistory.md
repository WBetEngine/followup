# Dokumentasi Halaman History Penanganan CRM

## Pendahuluan
Halaman History Penanganan CRM adalah fitur yang menampilkan riwayat lengkap penanganan member oleh CRM, termasuk riwayat rotasi, deposit, dan performa followup. Halaman ini membantu CRM, Admin, dan Superadmin untuk menganalisis dan meningkatkan efektivitas followup.

## Fitur Utama

1. **Riwayat Penanganan Member**
   - Daftar CRM yang pernah menangani member
   - Urutan penanganan (CRM 1 → CRM 2 → CRM 3 → ...)
   - Durasi penanganan per CRM
   - Status penanganan (berhasil/tidak berhasil)
   - Tanggal mulai dan selesai penanganan

2. **Statistik Deposit**
   - Jumlah deposit yang berhasil (approved)
   - Jumlah deposit yang ditolak (rejected)
   - Total nilai deposit yang berhasil
   - Total nilai deposit yang ditolak
   - Rata-rata nilai deposit per CRM
   - Persentase keberhasilan deposit per CRM

3. **Riwayat Rotasi**
   - Alasan rotasi (tidak deposit dalam 2 hari/dll)
   - Tanggal rotasi
   - CRM sebelumnya dan CRM baru
   - Status member saat dirotasi
   - Riwayat lengkap rotasi (CRM 1 → 2 → 3 → ... → 30 → 1)

4. **Analisis Performa**
   - Performa per CRM:
     - Jumlah member yang ditangani
     - Jumlah deposit berhasil
     - Jumlah deposit ditolak
     - Rata-rata waktu followup
     - Persentase keberhasilan
   - Performa per member:
     - Jumlah CRM yang menangani
     - Jumlah deposit berhasil
     - Jumlah deposit ditolak
     - Rata-rata nilai deposit

## Tampilan Data

1. **Tabel History**
   - Kolom yang ditampilkan:
     - Username member
     - CRM yang menangani
     - Tanggal penanganan
     - Status deposit
     - Nilai deposit
     - Alasan rotasi
     - Durasi penanganan

2. **Filter dan Pencarian**
   - Filter berdasarkan:
     - Tanggal
     - CRM
     - Status deposit
     - Nilai deposit
   - Pencarian berdasarkan:
     - Username member
     - Nomor CRM
     - Status

3. **Visualisasi Data**
   - Grafik performa CRM
   - Grafik statistik deposit
   - Grafik rotasi member
   - Grafik keberhasilan followup

## Akses dan Pembatasan

1. **Akses Berdasarkan Role**
   - CRM:
     - Hanya dapat melihat data member yang pernah ditangani
     - Dapat melihat statistik performa sendiri
   - Admin:
     - Dapat melihat semua data
     - Dapat melihat statistik semua CRM
   - Superadmin:
     - Akses penuh ke semua data
     - Dapat melihat statistik detail

2. **Pembatasan Data**
   - Data disimpan selama 1 tahun
   - Data lama dapat diarsipkan
   - Export data terbatas untuk Admin dan Superadmin

## Integrasi

1. **Integrasi dengan Sistem Rotasi**
   - Data rotasi otomatis tercatat
   - Riwayat penanganan CRM terupdate
   - Statistik performa terupdate

2. **Integrasi dengan Sistem Deposit**
   - Data deposit tercatat
   - Status deposit terupdate
   - Statistik deposit terupdate

3. **Integrasi dengan Sistem Notifikasi**
   - Notifikasi perubahan status
   - Notifikasi rotasi
   - Notifikasi performa

## Manfaat

1. **Untuk CRM**
   - Menganalisis performa followup
   - Meningkatkan strategi followup
   - Memahami pola keberhasilan

2. **Untuk Admin**
   - Monitoring performa CRM
   - Evaluasi sistem rotasi
   - Pengambilan keputusan

3. **Untuk Superadmin**
   - Analisis sistem secara keseluruhan
   - Pengembangan strategi
   - Optimasi sistem rotasi 