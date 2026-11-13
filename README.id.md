# Pigura

**Viewer foto & video self-hosted yang ringan**, ala Google Photos, dibuat untuk perangkat kecil (STB ARM,
Raspberry Pi, laptop lama, NAS). Satu binary Go statis + SQLite + PWA Svelte 5. Tanpa server database,
tanpa machine learning, RAM sekitar 20 MB saat idle (lihat [Performa](#performa)).

*Pigura* = bingkai foto. [Read in English](README.md)

> Pigura adalah proyek independen, **tidak berafiliasi** dengan Immich, Google, maupun Google Photos.
> Ia bisa mendeteksi otomatis folder `library/` bergaya Immich di disk, tetapi tidak pernah terhubung ke atau mengubah Immich.

## Tangkapan layar

<table>
  <tr>
    <td><img src="docs/screenshots/timeline.png" alt="Timeline grouped by day"><br><sub>Timeline, dikelompokkan per hari</sub></td>
    <td><img src="docs/screenshots/lightbox.png" alt="Photo lightbox"><br><sub>Lightbox: zoom, favorit, unduh, bagikan</sub></td>
  </tr>
  <tr>
    <td><img src="docs/screenshots/explore.png" alt="Explore view"><br><sub>Jelajah: filter jenis atau tanggal</sub></td>
    <td><img src="docs/screenshots/albums.png" alt="Albums"><br><sub>Albums</sub></td>
  </tr>
</table>

<sub>Wajah dan handle pihak ketiga sengaja diburamkan.</sub>

## Fitur

- Timeline cepat (keyset pagination), tampilan grid/daftar, filter tanggal, dan jelajah
- Album, favorit, lightbox dengan info EXIF, putar video (HTTP Range)
- Thumbnail & preview dibuat sesuai kebutuhan lalu di-cache (libvips, libheif untuk HEIC, ffmpeg untuk video)
- Scan library inkremental + auto-scan berkala (cocok dengan Syncthing / sinkronisasi ponsel)
- Pencari duplikat persis (ukuran → hash sampel → SHA-1 penuh)
- Tautan berbagi lewat **listener publik terpisah**, sehingga hanya item yang dibagikan yang terekspos
- PWA yang bisa dipasang, dengan cache thumbnail offline
- File asli hanya dibaca: "hapus" hanya menyembunyikan foto di Pigura, file di disk tidak disentuh
- Keamanan berlapis: allowlist host (anti DNS-rebinding), same-origin, login token opsional, CSP, antrean terbatas

Status: awal (v0.x). UI saat ini berbahasa Indonesia; kontribusi i18n sangat diterima.

> [!WARNING]
> **Disclaimer keamanan.** Pigura masih tahap awal (v0.x), disediakan "apa adanya" di bawah lisensi MIT, tanpa garansi.
>
> - **Tidak ada akun pengguna.** Kontrol akses hanya satu token bersama yang *opsional* (`PIGURA_TOKEN`). **Jika tidak
>   diisi, siapa pun yang bisa menjangkau port-nya dapat melihat, menyembunyikan, dan membagikan fotomu.** Selalu isi,
>   kecuali Pigura hanya bisa diakses dari `127.0.0.1`, misalnya: `export PIGURA_TOKEN=$(openssl rand -hex 32)`.
>   Di `docker-compose.yml` baris token dikomentari secara default; hapus tanda komentarnya.
> - **Jangan ekspos port utama langsung ke internet.** Pakai reverse proxy ber-TLS atau jaringan privat (VPN, Tailscale,
>   dll). Publikasikan hanya listener tautan berbagi (`PIGURA_PUBLIC_ADDR`) bila butuh tautan publik.
> - Login tidak bisa dicabut satu per satu dan belum ada logout: satu-satunya cara membatalkan sesi adalah mengganti token.
> - Pigura men-decode media tak tepercaya lewat libvips, libheif, dan ffmpeg. Perbarui ketiganya dan jalankan Pigura sebagai
>   user tanpa hak istimewa dengan library di-mount read-only.
> - Pigura tidak pernah mengubah atau menghapus file aslimu, tetapi ini **bukan backup**. Simpan backup sendiri.

## Mulai cepat (Docker)

```sh
git clone https://github.com/marzukiberg/pigura && cd pigura
# edit docker-compose.yml: isi path foto dan PIGURA_TOKEN
docker compose up -d
# buka http://127.0.0.1:8080
```

Foto di-mount read-only ke `/media/photos`; indeks dan cache thumbnail ada di volume `pigura-data`.

## Mulai cepat (binary)

```sh
./pigura -library /srv/photos -scan-base /srv -scan
```

Library harus berada **di dalam** `-scan-base` (bukan sama dengannya). Tool thumbnail opsional tapi disarankan:
`vipsthumbnail`, `heif-convert`, `ffmpeg`.

## Build dari source

Butuh Go (lihat `go.mod`) dan Node 22+. Frontend tertanam di binary, jadi build frontend dulu:

```sh
make build    # frontend + binary -> bin/pigura
make arm64    # binary statis Linux ARM64
make test
```

Konfigurasi lengkap (flag/env), model keamanan, dan unit systemd contoh: lihat [README.md](README.md),
bagian *Configuration*, *Security model*, dan `deploy/pigura.service`.

Perangkat RAM kecil: set `GOMEMLIMIT=80MiB` dan `VIPS_CONCURRENCY=2`.

## Performa

Diukur di STB ARM64 kelas bawah (Armbian, 4 core, RAM 1,8 GB yang dipakai bersama layanan lain, HDD putar) dengan library
nyata berisi 5.476 item (2.570 foto HEIC, 1.362 video), lewat loopback sehingga kecepatan jaringan tidak ikut terhitung.
"Cold" berarti thumbnail belum ada di cache; thumbnail hanya dibuat sekali lalu disajikan dari disk.

| Situasi | Hasil |
|---|---|
| Proses server saat idle | 13–18 MB RSS |
| Proses server selama seluruh benchmark | 18 MB RSS |
| Thumbnail JPEG cold | ~53 MB puncak (server + libvips), ~0,5 dtk |
| Thumbnail HEIC cold | ~121 MB puncak, ~4,3–4,9 dtk |
| Thumbnail video HEVC 1080p cold | ~145–150 MB puncak, 1,8–3,6 dtk |
| 8 request HEIC cold sekaligus | puncak tetap ~121 MB; diproses satu per satu, selesai ~33 dtk |
| Halaman timeline (~30 KB) | p50 9 ms, p95 10 ms |
| Thumbnail ter-cache (4 paralel) | p50 5 ms, p95 44 ms, ~60 req/dtk |

Artinya: **melayani** foto tetap sekitar 20 MB. Lonjakan ke ~120–150 MB hanya sesaat dari decoder eksternal dan dibatasi
`PIGURA_THUMB_WORKERS`, jadi layar penuh foto belum ter-cache akan antre, bukan memakai lebih banyak memori. Konsekuensinya,
konversi HEIC lambat di CPU lemah, tetapi hanya terjadi saat foto pertama kali ditampilkan. Satu kali jalan di satu perangkat:
indikasi, bukan benchmark ketat.

## Lisensi

[MIT](LICENSE).
