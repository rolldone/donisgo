# Changelog

Semua perubahan penting didokumentasikan di sini. Format mengikuti
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), versi mengikuti
[SemVer](https://semver.org/). Karena donisgo masih di jalur `v0.x`, perubahan
yang tidak kompatibel dapat muncul pada bump **minor**.

## [0.2.0] — 2026-11-25

Entri ini menggabungkan de-globalisasi `auth` (SPEC-004 D3) dan pembersihan
pure library (SPEC-005 fase 4).

### Removed (BREAKING)

- **`plugins/pages`** — plugin konten (serving SPA `/app`, redirect `/spa`,
  proxy `HOMEPAGE_URL`, wildcard `NoRoute`) dipindahkan ke
  `github.com/rolldone/donisgo-template/plugins/pages`.
- **`cmd/server` dan `cmd/console`** — donisgo kini nol binary; entrypoint
  aplikasi ada di `donisgo-template/cmd/*`.
- **Command `plugin new`** (implementasi di paket `console` + testnya) —
  generator scaffold aplikasi dipindahkan ke `donisgo-template/console/`.
- **`internal/pluginloader`** — dihapus sepenuhnya (`RegisterCorePlugins*`,
  `RegisterLastPlugins*`). Registrasi plugin kini 100% tanggung jawab pemanggil
  via `app.Config.Plugins`; donisgo tidak lagi mendaftarkan plugin apa pun.
- Dokumentasi app-level (quickstart, plugin dev, env, deploy) dan
  `.env.example` — dipindahkan ke repo starter.

### Added

- `auth.Config` dan `auth.New(cfg) (*Auth, error)` — API instans untuk paket
  `auth`; token ditandatangani/diverifikasi per instance sehingga dua instance
  dengan secret berbeda sepenuhnya terisolasi. `New` menolak secret kosong
  (tanpa default insecure). TTL opsional lewat `AccessTTL`/`RefreshTTL`/
  `ResetTTL` (nol → default 10m/14d/1h). `auth.ConfigFromEnv()` memetakan
  variabel lingkungan lama (`AUTH_JWT_SECRET` dulu, lalu `JWT_ACCESS_SECRET`/
  `JWT_REFRESH_SECRET`/`JWT_RESET_SECRET` + `JWT_*_EXP_SECONDS`) ke `Config`.
  (SPEC-004 D3)

### Changed (BREAKING)

- **Serving halaman/SPA tidak lagi otomatis.** `app` tidak menyajikan `/`,
  `NoRoute` catch-all, maupun `StaticFS`; `registerStoreRoutes` tetap ada
  sebagai titik force tetapi kosong. Aplikasi yang butuh serving halaman
  mendaftarkan plugin `pages` secara eksplisit.
- `app.attachPlugins` tidak lagi menambahkan plugin framework: urutan registry
  sepenuhnya ditentukan pemanggil. Komentar `plugins.Registry.EnsurePluginLast`
  diperbarui (kini mekanisme umum untuk plugin catch-all, bukan "paten
  donisgo").
- `console.RootCmd` memuat `migrate` & `seed` tanpa `plugin new`;
  `NewRootCmd()` tetap tersedia agar starter dapat menempelkan command sendiri.

### Deprecated

- API package-level `auth` (`Sign*Token`, `Parse*Token`,
  `GenerateAccessTokenWithLevel`, `ParseAccessTokenClaims`,
  `AccessExpirySeconds` dkk.) kini wrapper ke instance default-env dan ditandai
  `// Deprecated:`; dipertahankan hingga v1.0 (SPEC-004 D4). Perilaku lama tidak
  berubah.

### Documentation

- **Godoc lengkap untuk 12 paket publik** (`app`, `plugins`,
  `plugins/plugin_registry`, `db`, `events`, `keydb`, `auth`, `mail`, `storage`,
  `secrets`, `uuid`, `console`): setiap paket memiliki package doc
  (`doc.go`, konvensi `Package <name> ...`), seluruh simbol ekspor memiliki doc
  comment yang diawali nama simbol, dan `app.Config` serta `auth.Config`
  didokumentasikan per field. Semua dalam bahasa Inggris, siap render di
  [pkg.go.dev](https://pkg.go.dev/github.com/rolldone/donisgo). (SPEC-005 WO-06)
- **README dirombak menjadi README versi library** (Inggris): badge *Go
  Reference* + *MIT*, one-liner, install & pin, contoh `app.New`, tabel peta
  paket, catatan Ekosistem + serving `pages`, dan bagian versioning/releases.
- Komentar berbahasa Indonesia pada paket publik (`storage`, `mail`, `keydb`,
  `db`, `plugins`) diterjemahkan ke Inggris untuk konsistensi godoc.

### Migration notes

Dari `v0.1.x` ke `v0.2.0`:

1. **Aplikasi/sub-app yang mengandalkan serving halaman otomatis WAJIB
   mendaftarkan `pages` manual** dari repo starter, di urutan terakhir:

   ```go
   import pagesplugin "github.com/rolldone/donisgo-template/plugins/pages"

   application, err := app.New(app.Config{
       Plugins: []plugins.Plugin{myplugin.New()},
   })
   if err != nil {
       log.Fatal(err)
   }
   // jamin pages paling akhir (memegang router.NoRoute):
   application.GetRegistry().EnsurePluginLast(pagesplugin.New())
   ```

   Tanpa langkah ini, `/app` dan fallback SPA akan mengembalikan 404.

2. **Binary server & console**: pindahkan `cmd/server`/`cmd/console` dari
   salinan lokal ke repo `donisgo-template` (path sama), lalu sesuaikan import
   plugin aplikasi ke module path aplikasi Anda sendiri.

3. **Generator `plugin new`**: kini tersedia lewat CLI di repo starter
   (`go run ./cmd/console plugin new --id ...`). Hasil scaffold tetap mengimpor
   `github.com/rolldone/donisgo/...` untuk paket framework.

4. **Registrasi plugin**: tidak ada lagi `pluginloader`. Daftarkan plugin
   eksplisit via `app.Config.Plugins` (server) dan
   `console.RegisterAdditionalPlugins` (CLI).

5. **Referensi dokumentasi aplikasi** (quickstart, env, deploy, `.env.example`)
   pindah ke README repo starter.

## [0.1.4] — 2026-11-25

### Fixed

- `plugin new`: scaffold kini memisahkan path modul **framework** dari path
  modul **aplikasi**. Import paket framework (`plugins`, `events`) selalu
  memakai `github.com/rolldone/donisgo/...`, sehingga scaffold yang dihasilkan
  di dalam modul konsumen (mis. `historic_service`) dapat di-build. Sebelumnya
  seluruh import memakai module path dari `go.mod` CWD, sehingga import
  framework menjadi `<module-konsumen>/plugins` dan gagal build.
  Import paket milik aplikasi (`plugins/<id>/handlers`, `.../middleware`,
  `.../services`) tetap mengikuti module path aplikasi.
  ([#13](https://github.com/rolldone/donisgo/issues/13))

## [0.1.3] — 2026-11-25

### Added

- `CHANGELOG.md` (file ini) + section "Versioning & releases" di README:
  panduan pin versi (`go get ...@v0.1.2`), caveat breaking change pada bump
  minor jalur `v0.x`, dan rujukan ke tag GitHub. (PR #12, commit 934902c)

## [0.1.2] — 2026-11-25

### Changed

- `console` dipromosikan menjadi paket publik
  `github.com/rolldone/donisgo/console` (sebelumnya `internal/console`), sehingga
  sub-app dan tooling lain dapat memakai ulang command CLI framework.
  `internal/` kini hanya berisi `internal/pluginloader`.
  (commit `04bd40b`)

## [0.1.1] — 2026-11-25

### Changed

- Env `TRUSTED_PROXIES` (comma-separated IP/CIDR) di-wire ke gin
  `SetTrustedProxies`, memberi kontrol eksplisit atas `ClientIP()`.

### Security

- **PERUBAHAN PERILAKU** — default sekarang *trust-none*: tidak ada proxy yang
  dipercaya, sehingga `ClientIP()` mengembalikan `RemoteAddr`. Aplikasi yang
  berjalan di belakang reverse proxy **WAJIB** mengisi `TRUSTED_PROXIES`, jika
  tidak semua request akan terlihat berasal dari IP proxy.
  (issue #3, commit `1d7ea47`)

## [0.1.0] — 2026-11-25

### Changed (BREAKING)

- SPEC-004: 10 paket dipindahkan dari `internal/` ke root modul — `app`,
  `plugins`, `db`, `storage`, `mail`, `auth`, `secrets`, `uuid`, `events`,
  `keydb`. Semua import `github.com/rolldone/donisgo/internal/...` untuk paket
  tersebut **hilang** dan harus diganti ke path root (mis.
  `internal/app` → `github.com/rolldone/donisgo/app`).
- Wiring kini injektable: `app.Config` + `app.New(cfg)`, `plugins.Registry`
  instance-scoped, serta API instans `db.Open`, `events.NewBus`, `keydb.New`.
  Fungsi package-level lama tetap ada sebagai wrapper `Deprecated:`.
- `plugin new`: scaffold yang dihasilkan kini gofmt-clean dan tidak lagi memuat
  literal `\n` di dalam file Go (fix #2).
- Contoh konsumsi sebagai library ditambahkan di
  [`examples/library/`](examples/library) — modul terpisah dengan direktif
  `replace`.

(commit `56af15b`)

[0.2.0]: https://github.com/rolldone/donisgo/compare/v0.1.4...v0.2.0
[0.1.4]: https://github.com/rolldone/donisgo/compare/v0.1.3...v0.1.4
[0.1.3]: https://github.com/rolldone/donisgo/releases/tag/v0.1.3
[0.1.2]: https://github.com/rolldone/donisgo/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/rolldone/donisgo/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/rolldone/donisgo/releases/tag/v0.1.0
