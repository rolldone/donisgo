# SPEC-004 — Donisgo sebagai Library yang Bisa Diimpor

| | |
|---|---|
| ID | SPEC-004 |
| Status | **APPROVED — ACC pemilik repo, 2026-11-25** |
| Ref issue | https://github.com/rolldone/donisgo/issues/4 |
| WO terkait | WO-01 topic `1795575759044` (status: blocked, terbuka setelah SPEC ini ACC) |
| Penyusun | Manager Agent |
| Tanggal | 2026-11-25 |
| Branch | `spec/4-library` |

---

## 1. Latar Belakang & Masalah

Semua paket inti framework berada di `internal/`: `app`, `auth`, `console`, `db`,
`events`, `keydb`, `mail`, `pluginloader`, `plugins`, `secrets`, `storage`, `uuid`.
Aturan Go melarang import `internal/` dari modul lain, sehingga proyek eksternal
tidak dapat `import "github.com/rolldone/donisgo/..."`. Donisgo saat ini hanya
bisa dipakai sebagai scaffold aplikasi, bukan library.

Selain itu sebagian wiring masih global-state (detail di §4-D3), sehingga dua
instans `App` dalam satu proses tidak mungkin dan pengujian terisolasi sulit.

## 2. Tujuan

1. Paket inti bisa diimpor dari modul eksternal apa pun (`go get github.com/rolldone/donisgo`).
2. Wiring jadi injektable: konfigurasi eksplisit, tanpa global state pada jalur bootstrap utama.
3. Perilaku runtime tidak berubah — ini refactor murni, bukan fitur baru.
4. Scaffold `plugin new` dan plugin built-in tetap berfungsi dengan path baru.

## 3. Non-Goals

- Tidak menambah fitur baru.
- Tidak menyentuh `sub_app/webapp` (frontend Vite — bukan Go, tidak terdampak).
- Tidak mengubah backend storage/DB yang didukung.
- Tidak menjanjikan stabilitas API v1 — lihat strategi versi (§4-D5).

## 4. Keputusan Desain (D1–D6)

### D1 — Layout: paket publik di root modul (bukan `pkg/`)

`internal/<pkg>` → `<pkg>` di root modul (mis. `internal/db` → `db`, import jadi
`github.com/rolldone/donisgo/db`).

**Alasan**: idiomatik untuk library, import pendek, sesuai gaya repo (sudah ada
`plugins/` di root untuk plugin built-in). Alternatif `donisgo/pkg/db` ditolak:
tidak menambah nilai, hanya menambah panjang import.

### D2 — Pembagian paket: publik vs tetap internal

| Paket | Keputusan | Catatan |
|---|---|---|
| `plugins` (kontrak) | **PUBLIK** | Jantung library: `Plugin`, `ServiceDeps`, `MiddlewareDescriptor`. Kontrak interface TIDAK berubah bentuk, hanya path import. |
| `db` | **PUBLIK** | `WithTransaction` langsung layak publik; `GetDB`/`GetGormDB` jadi deprecated (D3). |
| `storage` | **PUBLIK** | `Store`, `Config`, `NewStore` sudah bersih; `LoadConfig` tetap sebagai helper env. |
| `mail` | **PUBLIK** | `Mailer`, `Mailable`, `SMTPConfig`, `NewMailerWithConfig` sudah injektable. |
| `auth` | **PUBLIK** | Fungsi JWT murni; de-globalisasi baca env menyusul (D3). |
| `secrets` | **PUBLIK** | `EncryptBlob`/`DecryptBlob`; `MasterKey()` tetap helper env. |
| `uuid` | **PUBLIK** | Pure function, tanpa state — promosi termurah. |
| `events` | **PUBLIK** | Bus pub/sub; de-globalisasi via `events.Bus` (D3). |
| `keydb` | **PUBLIK** | `Flash` + client; `var Client` global dihapus (D3). |
| `app` | **PUBLIK (fase akhir)** | Wajah wiring: `app.Config` + `app.New`. Dieksekusi terakhir karena paling banyak digali. |
| `console` | **PUBLIK** (amendemen 2026-11-25) | CLI tooling `cmd/console` — dipromosikan sejak `v0.1.2` (`04bd40b`): sub-app & tooling lain memakai ulang command CLI framework. Lihat Log Amendemen. |
| `pluginloader` | internal | Wiring registry plugin — dipanggil dari `app`, bukan API publik. |

### D3 — Wiring injektable: hapus global state

Global state yang teridentifikasi (hasil audit) dan penggantinya:

| Global sekarang | Pengganti |
|---|---|
| `db.GetGormDB()` (singleton env) | `db.Open(cfg)` → `*gorm.DB` milik pemanggil; env helper tetap ada untuk CLI/scaffold |
| `keydb.Init(...)` + `var keydb.Client` | `keydb.New(cfg) (*Client, error)` — instance dimiliki `App` |
| Bus global `events.Publish/Subscribe` | `events.NewBus() (*Bus)`; `App` memegang bus; fungsi package-level deprecated (wrapper ke bus default) selama masa transisi |
| Registry global `plugins.RegisterPlugins/RegisteredPlugins` | `plugins.Registry` (instance per `App`); `AttachMiddleware/RegisterAllRoutes/RegisterAllServices/SeedAll` jadi method registry |
| `auth.*` membaca env saat dipanggil | Tambah `auth.Config` (secret, TTL) opsional; wrapper default-env dipertahankan sementara, ditandai deprecated |
| `secrets.MasterKey()` env | `secrets.NewKey([]byte)`; helper env tetap ada |
| `storage.LoadConfig()` env | Tetap ada sebagai helper; `App` menerima `*storage.Config` eksplisit |

`app.New(opts)` yang sekarang (`Options.RegisterPlugins func()`) diperluas jadi:

```go
cfg := app.Config{
    DB:              *gorm.DB,          // opsional; nil → db.Open dari env
    Store:           storage.Store,     // opsional; nil → NewStore(LoadConfig())
    KeyDB:           *keydb.Client,     // opsional
    Mail:            *mail.Mailer,      // opsional
    TrustedProxies:  []string{...},     // opsional; nil → aman (tidak percaya proxy)
    CORS:            *cors.Config,      // opsional
    Plugins:         []plugins.Plugin{myplugin.New()},
}
application, err := app.New(cfg)    // tanpa env wajib
err = application.Run()
```

`app.NewFromEnv()` tetap disediakan sebagai gula untuk perilaku lama
(env-driven) — dipakai oleh `cmd/server` & `cmd/console`.

### D4 — Kompatibilitas selama transisi

- Wrapper default-env (deprecated) dipertahankan di paket `db`, `events`, `auth`,
  `secrets` hingga v1.0, agar scaffold lama yang sudah menyebar tidak langsung putus.
- Import path `internal/...` **tidak** diberi alias — dihapus langsung. Alasan:
  repo ini pre-1.0 dan konsumen utamanya adalah scaffold buatannya sendiri
  (`plugin new`) yang ikut diperbarui di fase yang sama. (Menunggu konfirmasi
  di Open Question OQ-3.)

### D5 — Strategi versi

- Semua perubahan ini rilis sebagai **`v0.x`** (diawali tag `v0.1.0` saat SPEC
  selesai dieksekusi). Breaking change masih diperbolehkan di v0 dengan
  increment minor + catatan rilis.
- Stabilitas janji v1.0 baru dipublikasikan setelah API terpakai dan uji.
- Setiap fase (D6) = satu commit terpisah dengan gerbang kualitas penuh.

### D6 — Urutan eksekusi (5 fase)

| Fase | Isi | Risiko |
|---|---|---|
| 1 | Promosi mekanis paket tanpa ubah API: `uuid`, `auth`, `secrets`, `mail`, `storage` | Rendah |
| 2 | `db`, `events`, `keydb` + API instans baru (`db.Open`, `events.NewBus`, `keydb.New`), global deprecated | Sedang |
| 3 | `plugins` (kontrak + `Registry` instance-scoped) + `pluginloader` mengikuti | Sedang |
| 4 | `app` publik + `app.Config` + refactor bootstrap; `cmd/*` makin thin | Tinggi |
| 5 | Scaffold `plugin new` (path baru), README "Using donisgo as a library", contoh modul eksternal, tag `v0.1.0` | Rendah |

Gerbang kualitas tiap fase: `gofmt -l .` kosong, `go vet ./...`, `go build ./...`,
`go test ./...` hijau, commit pesan eksplisit per fase.

### Log Amendemen

| Tanggal | Keputusan | Alasan & bukti |
|---|---|---|
| 2026-11-25 | **D2** — `console`: internal → **PUBLIK** | Dipromosikan sejak rilis `v0.1.2` (commit `04bd40b`): sub-app & tooling lain perlu memakai ulang command CLI framework; `internal/` kini hanya `pluginloader` — tetap internal (lem wiring, konsumennya hanya `app` & `console` di dalam modul). Disetujui pemilik repo. (Historic topic `1795594944725`, issue #16) |

## 5. Dampak & Migrasi

- **Plugin built-in** (`plugins/pages`, `plugins/plugin_registry` di root):
  import path `internal/plugins` → `plugins` — ikut fase 3.
- **Scaffold `plugin new`**: template menulis `"%s/internal/plugins"` (dan
  import handlers plugin) — path baru dipakai di fase 5; sampai fase itu,
  scaffold bekerja seperti sekarang.
- **`sub_app/webapp`**: tidak terdampak.
- **Deployment**: tidak ada perubahan env/behavior → tanpa aksi operator.

## 6. Risiko & Mitigasi

| Risiko | Mitigasi |
|---|---|
| Breaking change besar (hapus `internal/`) | v0 + semua konsumen internal diperbarui di commit yang sama; catatan rilis jelas |
| Dua instans `App` belum benar-benar bebas global saat transisi | Wrapper deprecated eksplisit + daftar global tersisa di catatan tiap fase; v1.0 baru dijanjikan setelah nol global |
| Regresi perilaku | Refactor murni: tidak ada perubahan logika; test existing + test baru per fase |
| Repo fork lama (`go_framework`) lewat dari path lama | Tidak dibenahi — hanya repo ini yang diformat ulang |

## 7. Acceptance Criteria (SPEC ini)

- [x] ACC atas keputusan D1–D6 dan jawaban atas OQ-1..OQ-4 (§8) — **ACC 2026-11-25: semua OQ mengikuti rekomendasi.**
- [ ] WO-01 topic `1795575759044` dibuka: `blocked → planned`, checklist fase jadi acuan eksekusi.
- [ ] Bukti akhir eksekusi (fase 5): modul eksternal contoh `go get` + import hijau.

## 8. Open Questions (butuh jawaban pemilik repo)

> **RESOLVED 2026-11-25** — pemilik repo memilih mengikuti seluruh rekomendasi penyusun.

- **OQ-1** — Path publik root-level (rekomendasi saya) atau pakai grup `pkg/`? → **Root-level** ✅
- **OQ-2** — `app` ikut dipublikasikan di refactor ini (fase 4) atau ditunda setelah v0 pertama? → **Ikut, fase 4** ✅
- **OQ-3** — Alias `internal/` dihapus langsung tanpa masa alias (rekomendasi saya, pre-1.0) atau perlu masa transisi? → **Hapus langsung** ✅
- **OQ-4** — Tag `v0.1.0` saat semua fase selesai — disetujui? → **Ya** ✅
