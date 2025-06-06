# Dokumentasi Sistem Rotasi Member di Halaman Followup

## Pendahuluan
Sistem rotasi member adalah fitur yang mengatur perpindahan member antar CRM di halaman followup berdasarkan jadwal followup dan performa. Sistem ini memastikan setiap member mendapatkan followup yang optimal dan merata di halaman followup.

## Fitur Utama

1. **Jadwal Followup di Halaman Followup**
   - Maksimal waktu followup: 2 hari per CRM
   - Perhitungan waktu dimulai saat member pertama kali ditangani CRM di halaman followup
   - Sistem akan mencatat:
     - Tanggal mulai followup
     - CRM yang menangani
     - Status followup (aktif/pending/rotasi)
   - Timer hanya berlaku untuk member di halaman followup

2. **Sistem Rotasi di Halaman Followup**
   - Trigger rotasi otomatis oleh sistem:
     - Member tidak melakukan deposit dalam 2 hari di halaman followup
     - Member belum di-followup dalam 2 hari di halaman followup
   - Proses rotasi:
     - Sistem secara otomatis memindahkan member ke CRM berikutnya di halaman followup
     - Rotasi dilakukan secara berurutan (1,2,3,...,30,1,2,3,...)
     - Sistem mencatat riwayat CRM yang pernah menangani member
     - Riwayat rotasi dicatat dalam sistem
   - Rotasi dilakukan tanpa intervensi manual
   - Sistem akan memilih CRM baru secara otomatis berdasarkan urutan
   - Rotasi hanya berlaku untuk member di halaman followup

3. **Penentuan CRM Baru untuk Halaman Followup**
   - Kriteria pemilihan CRM:
     - Rotasi berurutan berdasarkan nomor CRM (1,2,3,...,30)
     - Jika sudah sampai CRM ke-30, kembali ke CRM ke-1
     - Sistem mencatat urutan CRM yang sudah menangani member
     - CRM yang sedang cuti/off akan dilewati
   - Sistem akan menghindari:
     - CRM yang sedang cuti/off
     - CRM yang memiliki beban member terlalu banyak di halaman followup
   - Contoh rotasi:
     - Member A: CRM 1 → CRM 2 → CRM 3 → ... → CRM 30 → CRM 1
     - Member B: CRM 2 → CRM 3 → CRM 4 → ... → CRM 1 → CRM 2
     - Member C: CRM 3 → CRM 4 → CRM 5 → ... → CRM 2 → CRM 3

4. **Notifikasi Rotasi**
   - Notifikasi ke CRM lama:
     - Pemberitahuan member akan dirotasi di halaman followup
     - Alasan rotasi
     - Waktu rotasi
     - Nomor urut CRM berikutnya
   - Notifikasi ke CRM baru:
     - Pemberitahuan member baru di halaman followup
     - Data member
     - Riwayat followup sebelumnya
     - Riwayat CRM yang pernah menangani
   - Notifikasi ke Admin:
     - Laporan rotasi di halaman followup
     - Statistik performa CRM
     - Distribusi beban member per CRM

## Alur Kerja

1. **Proses Followup di Halaman Followup**
   - CRM mendapatkan member baru di halaman followup
   - Sistem mencatat waktu mulai followup
   - CRM melakukan followup sesuai jadwal
   - Sistem memantau aktivitas followup

2. **Proses Rotasi di Halaman Followup**
   - Sistem mengecek status followup setiap hari
   - Jika kondisi rotasi terpenuhi:
     - Sistem menentukan CRM berikutnya berdasarkan urutan
     - Memindahkan member ke CRM baru di halaman followup
     - Mengirim notifikasi ke semua pihak
     - Mencatat riwayat rotasi dan urutan CRM
     - Jika sudah sampai CRM ke-30, kembali ke CRM ke-1

3. **Monitoring Halaman Followup**
   - Dashboard untuk melihat:
     - Member yang akan dirotasi di halaman followup
     - Performa CRM
     - Statistik rotasi
     - Riwayat rotasi
     - Distribusi beban member per CRM
     - Urutan CRM yang menangani setiap member

## Batasan dan Validasi

1. **Waktu Followup di Halaman Followup**
   - Maksimal 2 hari per CRM
   - Tidak bisa diperpanjang
   - Reset timer jika member melakukan deposit
   - Rotasi otomatis setelah 2 hari tanpa deposit
   - Timer dihitung sejak member pertama kali ditangani CRM di halaman followup

2. **Pembatasan CRM di Halaman Followup**
   - Rotasi otomatis jika member tidak melakukan deposit dalam 2 hari
   - CRM dengan beban member terlalu banyak tidak akan mendapat member baru
   - CRM yang sedang cuti/off tidak akan mendapat member baru
   - Rotasi dilakukan sepenuhnya oleh sistem tanpa intervensi manual
   - Tidak ada pembatasan waktu untuk CRM menangani member yang sama
   - CRM bisa menangani member yang sama jika member tersebut melakukan deposit

3. **Validasi Rotasi di Halaman Followup**
   - Rotasi hanya bisa dilakukan oleh sistem secara otomatis
   - CRM tidak bisa memindahkan member secara manual
   - Admin dapat melihat riwayat rotasi
   - Sistem mencatat alasan rotasi
   - Tidak ada opsi untuk membatalkan rotasi yang sudah dilakukan sistem
   - Rotasi hanya terjadi jika member tidak melakukan deposit dalam 2 hari
   - Rotasi hanya berlaku untuk member di halaman followup
   - Rotasi harus mengikuti urutan CRM (1-30)
   - Jika CRM sedang cuti/off, sistem akan melewati CRM tersebut
   - Setelah CRM ke-30, rotasi akan kembali ke CRM ke-1 