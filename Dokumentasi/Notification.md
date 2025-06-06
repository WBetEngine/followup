# Dokumentasi Fitur Notifikasi

## Pendahuluan
Sistem notifikasi digunakan untuk memberikan informasi real-time kepada pengguna tentang perubahan status deposit dan aktivitas penting lainnya.

## Jenis Notifikasi

1. **Notifikasi Deposit**
   - Notifikasi saat deposit dibatalkan
   - Notifikasi saat status deposit berubah (approve/reject)
   - Notifikasi untuk CRM saat deposit mereka dibatalkan/disetujui

2. **Notifikasi Status**
   - Perubahan status dari "New Deposit" ke "Redeposit"
   - Perubahan status deposit (Pending/Approved/Rejected)

## Integrasi

1. **Integrasi dengan Menu Followup**
   - Notifikasi muncul saat ada perubahan status deposit
   - Notifikasi untuk CRM tentang deposit yang mereka input

2. **Integrasi dengan Halaman Deposit**
   - Notifikasi untuk admin/superadmin tentang deposit yang perlu disetujui
   - Notifikasi untuk CRM saat deposit mereka disetujui/dibatalkan

3. **Integrasi dengan History Approve Deposit**
   - Notifikasi muncul di halaman notifikasi ketika:
     - Deposit yang sudah approved dibatalkan
     - Status deposit berubah (approve/reject)

## Tampilan Notifikasi

1. **Format Notifikasi**
   - Tampilan: Box dengan warna sesuai jenis notifikasi
   - Informasi yang ditampilkan:
     - Jenis notifikasi
     - Username member
     - Status
     - Waktu kejadian
     - Role yang melakukan perubahan

2. **Pengelompokan**
   - Notifikasi dikelompokkan berdasarkan jenis (approve/reject)
   - Notifikasi dikelompokkan berdasarkan waktu
   - Notifikasi terbaru muncul di bagian atas 