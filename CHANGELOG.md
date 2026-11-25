# Changelog

Semua perubahan penting didokumentasikan di sini. Format mengikuti
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), versi mengikuti
[SemVer](https://semver.org/). Karena donisgo masih di jalur `v0.x`, perubahan
yang tidak kompatibel dapat muncul pada bump **minor**.

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

[0.1.2]: https://github.com/rolldone/donisgo/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/rolldone/donisgo/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/rolldone/donisgo/releases/tag/v0.1.0
