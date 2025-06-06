# Dokumentasi Halaman Team

## Pendahuluan
Halaman Team adalah fitur untuk mengelola struktur team dalam sistem, termasuk pembuatan team, penambahan anggota, dan pengaturan akses. Halaman ini terintegrasi dengan halaman Langganan dan digunakan untuk mengatur pembatasan akses data.

## Fitur Utama

1. **Manajemen Team**
   - Pembuatan team baru
   - Penambahan anggota team
   - Penghapusan anggota team
   - Pengaturan admin team
   - Pengaturan nama team
   - Pengaturan deskripsi team

2. **Struktur Team**
   - Admin Team:
     - Satu admin per team
     - Bertanggung jawab atas team
     - Dapat mengelola anggota team
   - Anggota Team:
     - CRM yang tergabung dalam team
     - Dapat dirotasi antar team
     - Akses terbatas pada data team

3. **Akses Berdasarkan Role**
   - Superadmin:
     - Dapat membuat/menghapus team
     - Dapat menambah/menghapus anggota
     - Dapat mengatur admin team
     - Akses ke semua team
   - Admin Team:
     - Dapat melihat anggota team
     - Dapat menambah/menghapus anggota
     - Akses terbatas pada team sendiri
   - CRM:
     - Dapat melihat team sendiri
     - Tidak dapat mengubah struktur team
     - Akses terbatas pada data team

## Alur Kerja

1. **Pembuatan Team**
   - Superadmin membuat team baru
   - Mengatur nama dan deskripsi team
   - Menentukan admin team
   - Team siap digunakan

2. **Penambahan Anggota**
   - Admin/Superadmin menambah anggota
   - Memilih CRM yang akan ditambahkan
   - Mengatur role dalam team
   - Anggota terdaftar dalam team

3. **Pengaturan Akses**
   - Sistem mengecek role user
   - Menentukan akses berdasarkan team
   - Membatasi akses data sesuai team
   - Mencatat perubahan akses

## Integrasi

1. **Integrasi dengan Halaman Langganan**
   - Data team tercermin di halaman langganan
   - Pembatasan akses berdasarkan team
   - Statistik per team

2. **Integrasi dengan Halaman Followup**
   - Data followup dibatasi per team
   - Rotasi member dalam team
   - Statistik performa team

3. **Integrasi dengan Sistem Notifikasi**
   - Notifikasi perubahan team
   - Notifikasi penambahan anggota
   - Notifikasi perubahan akses

## Batasan dan Validasi

1. **Pembatasan Team**
   - Satu CRM hanya bisa dalam satu team
   - Satu admin hanya bisa mengelola satu team
   - Team tidak bisa kosong (minimal 1 admin)
   - Nama team harus unik

2. **Validasi Anggota**
   - CRM harus aktif
   - CRM tidak boleh dalam team lain
   - Role harus valid
   - Akses harus sesuai role

3. **Validasi Admin**
   - Admin harus memiliki role admin
   - Admin tidak bisa dalam team lain
   - Admin harus aktif
   - Akses sesuai dengan team 