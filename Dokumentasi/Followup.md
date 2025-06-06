# Dokumentasi Menu Followup



## Pendahuluan
Menu Followup adalah halaman yang berfungsi untuk mengelola dan memantau data followup member. Halaman ini menyediakan tampilan tabel data member yang diambil dari database members dengan berbagai informasi penting seperti username, email, nomor telepon, informasi bank, brand, status, dan CRM yang menangani.



## Tujuan
Menu Followup dibuat dengan beberapa tujuan utama:

1. **Manajemen Data Member**
   - Menampilkan data member dalam format tabel yang terstruktur
   - Menyediakan informasi lengkap tentang member termasuk:
     - ID (No)
     - Username
     - Email (membership_email)
     - Nomor Telepon (phone_number)
     - Informasi Bank (bank_name)
     - Nomor Rekening (account_no)
     - Brand (brand_name || currentBrandNameValue)
     - Status (Redopoist atau New Deposit)
     - CRM yang menangani (crmUsername)
  

2. **Manajemen Akses Berbasis Role**
   - Menerapkan sistem pembatasan akses berdasarkan role pengguna
   - Memastikan keamanan data dengan pembatasan akses yang tepat
   - Mencegah akses tidak sah ke data member

3. **Pemisahan Data Berdasarkan Role**
   - Role CRM: Hanya dapat melihat data member yang mereka tangani
   - Role Superadmin: Dapat melihat semua data member tanpa batasan
   - Role Admin: Hanya dapat melihat data member dari tim CRM yang bekerja sama
   - Role Adminstrator tidak bisa mengakses Halaman ini

4. **Efisiensi Kerja Tim**
   - Memudahkan koordinasi antara Admin dan tim CRM
   - Memungkinkan Admin untuk memantau kinerja tim CRM mereka
   - Memudahkan tracking progress followup member

## Pengguna
Menu Followup digunakan oleh beberapa role pengguna dengan tanggung jawab yang berbeda:

1. **Superadmin**
   - Memiliki akses penuh ke semua data member
   - Dapat melihat dan memantau semua aktivitas followup
   - Memiliki kendali penuh atas sistem

2. **Admin**
   - Dapat melihat data member dari tim CRM yang bekerja sama
   - Memantau kinerja tim CRM
   - Mengelola dan mengkoordinasikan tim CRM

3. **CRM**
   - Mengakses data member yang ditangani
   - Melakukan followup melalui WhatsApp
   - Mengelola proses deposit member
   - Menggunakan fitur copy untuk nomor telepon dan nomor rekening

4. **Tele Marketing**
   - Mendukung tim CRM dalam proses followup
   - Mengakses data member yang ditangani

5. **Administrator**
   - Tidak memiliki akses ke Menu Followup
   - Mengelola aspek administratif sistem lainnya

## Fitur Utama

1. **Tabel Data Member**
   - Menampilkan informasi lengkap member
   - Fitur copy untuk nomor telepon dan nomor rekening
   - Tombol Deposit untuk setiap baris data

2. **Fitur Copy**
   - Tombol SVG copy di samping nomor telepon
   - Tombol SVG copy di samping nomor rekening
   - Memudahkan CRM untuk mengirim pesan WhatsApp

3. **Proses Deposit**
   - Modal popup untuk input deposit
   - Field input:
     - Mata uang (Rp)
     - Nama
     - Role penginput
   - Status "Pending" di tabel setelah input
   - Integrasi dengan halaman Deposit

4. **Manajemen Status**
   - Status "New Deposit" untuk deposit pertama kali
   - Status "Redeposit" untuk deposit kedua dan seterusnya
   - Status "Redeposit" akan tetap "Redeposit" untuk semua deposit berikutnya
   - Perubahan status otomatis di database members

## Alur Kerja

1. **Proses Followup**
   - CRM mengakses data member
   - Menggunakan fitur copy untuk nomor kontak
   - Melakukan followup via WhatsApp

2. **Proses Deposit**
   - CRM mengklik tombol Deposit
   - Mengisi form deposit di modal popup
   - Status berubah menjadi "Pending"
   - Data dikirim ke halaman Deposit

3. **Persetujuan Deposit**
   - Deposit diproses di halaman Deposit
   - Setelah disetujui:
     - Status "Pending" hilang
     - Data masuk ke halaman Langganan
     - Jika status sebelumnya "New Deposit", berubah menjadi "Redeposit"
     - Jika status sebelumnya "Redeposit", tetap "Redeposit"

4. **Update Status**
   - Sistem otomatis mengupdate status di database members
   - Perubahan status tercermin di tabel followup

## Integrasi

1. **Integrasi dengan Halaman Deposit**
   - Data deposit dari Menu Followup dikirim ke halaman Deposit
   - Hanya Admin dan Superadmin yang dapat mengakses halaman Deposit
   - CRM tidak memiliki akses ke halaman Deposit
   - Proses persetujuan deposit dilakukan di halaman Deposit

2. **Integrasi dengan Halaman Langganan**
   - Data member yang depositnya disetujui akan dipindahkan ke halaman Langganan
   - Status member akan berubah dari "New Deposit" menjadi "Redeposit" (hanya untuk deposit pertama)
   - Status "Redeposit" akan tetap "Redeposit" untuk semua deposit berikutnya
   - Perubahan status otomatis di database members

3. **Alur Data Antar Menu**
   a. **Dari Followup ke Deposit**
      - CRM melakukan input deposit di Menu Followup
      - Status "pending" muncul di samping username (huruf kecil, box orange)
      - Data dikirim ke halaman Deposit untuk persetujuan
   
   b. **Dari Deposit ke Langganan**
      - Admin/Superadmin menyetujui deposit di halaman Deposit
      - Data dipindahkan ke halaman Langganan
      - Status diupdate menjadi "Redeposit"
      - Perubahan tercermin di database members

4. **Pembatasan Akses**
   - CRM: Hanya dapat mengakses Menu Followup
   - Admin: Dapat mengakses Menu Followup dan halaman Deposit
   - Superadmin: Dapat mengakses semua menu termasuk Deposit
   - Administrator: Tidak dapat mengakses Menu Followup

## Spesifikasi Teknis


1. **Error Handling**
   - Validasi input sebelum pengiriman data
   - Pesan error yang jelas untuk setiap validasi
   - Logging untuk setiap error yang terjadi

## Spesifikasi UI/UX

1. **Tampilan Tabel**
   - Pagination: 50 data per halaman
   - Sorting berdasarkan kolom
   - Filter berdasarkan status
   - Pencarian berdasarkan username/email/Nohp

2. **Status Pending**
   - Tampilan: Box orange dengan teks "pending" (lowercase)
   - Posisi: Di samping username
   - Style: Background-color: #FFA500, Color: white, Padding: 2px 8px

3. **Tombol Copy**
   - Icon: SVG copy
   - Posisi: Di samping nomor telepon dan nomor rekening
   - Feedback: Toast notification saat berhasil copy
   - Hover effect: Background color change

## Batasan dan Validasi

1. **Deposit**
   - Minimum deposit per transaksi: Rp 10.000

2. **Akses dan Keamanan**
   - Logging untuk setiap akses ke halaman deposit

3. **Pembatalan Deposit**
   - Hanya admin/superadmin yang dapat membatalkan
   - Deposit yang sudah approved bisa di batalkan
   - Notifikasi ke CRM saat deposit dibatalkan dan di setujui
   - Logging untuk setiap pembatalan dan setujui
