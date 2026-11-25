Donisgo — Go library untuk backend modular
==========================================

Summary
-------
**Donisgo** adalah **library** Go (bukan aplikasi) untuk membangun backend
modular: koneksi GORM, sistem migrasi, kontrak plugin, event bus, client
KeyDB/Redis, storage, mailer, dan utilitas JWT. Ia tidak mengirim binary,
tidak menyajikan halaman, dan tidak melakukan scaffolding aplikasi — semua itu
ada di repo starter
[`donisgo-template`](https://github.com/rolldone/donisgo-template).

Dokumen ini menjelaskan API library. Untuk quickstart aplikasi (server, CLI,
`plugin new`, env, deploy) lihat README donisgo-template.

Ekosistem
---------
| Repo | Peran | Cara konsumsi |
|---|---|---|
| **donisgo** (repo ini) | **Pure library** — paket inti yang bisa diimpor | `go get github.com/rolldone/donisgo@vX.Y.Z`, di-pin di `go.mod` |
| [`donisgo-template`](https://github.com/rolldone/donisgo-template) | **Starter aplikasi + tooling** — `cmd/server`, `cmd/console`, generator `plugin new`, plugin konten `pages`, skeleton frontend, docs aplikasi & `.env.example` | clone / fork / GitHub *Use this template* |

Aturan batas:

- **Library** berisi kontrak & infrastruktur: `app`, `plugins`, `db`, `auth`,
  `events`, `keydb`, `storage`, `mail`, `secrets`, `uuid`, `console`.
- **Starter** berisi wujud aplikasi: binary, scaffold, plugin konten.
- Sejak `v0.2.0` **serving halaman/SPA tidak lagi otomatis** dilakukan
  framework. Aplikasi yang butuh serving halaman wajib mendaftarkan plugin
  `pages` secara eksplisit dari
  `github.com/rolldone/donisgo-template/plugins/pages` (urutan terakhir via
  `plugins.Registry.EnsurePluginLast`).

Instalasi & versioning
----------------------
donisgo memakai [SemVer](https://semver.org/) dengan tag Git. Selalu pin versi:

```bash
go get github.com/rolldone/donisgo@v0.2.0
```

⚠️ Jalur `v0.x`: perubahan yang tidak kompatibel dapat muncul pada bump
**minor** (`v0.1.x` → `v0.2.0`). Baca [`CHANGELOG.md`](CHANGELOG.md) sebelum naik
versi. Riwayat rilis: tag GitHub.

Paket publik
------------
| Package | Import path | Yang didapat |
|---|---|---|
| app | `github.com/rolldone/donisgo/app` | `app.Config`, `app.New`, `app.NewFromEnv`, `(*App).Run`, `GetStore/GetMailer/GetKeyDB/GetRegistry` |
| plugins | `github.com/rolldone/donisgo/plugins` | Kontrak `Plugin`, `ServiceDeps`, `MiddlewareDescriptor`, `Registry`, konstanta prioritas |
| db | `github.com/rolldone/donisgo/db` | `db.Open(db.Config{...})`, `db.ConfigFromEnv()`, `WithTransaction` |
| events | `github.com/rolldone/donisgo/events` | `events.NewBus()` publish/subscribe + request/reply |
| keydb | `github.com/rolldone/donisgo/keydb` | `keydb.New(cfg)` → `*Client` (flash message, Redis mentah) |
| storage | `github.com/rolldone/donisgo/storage` | Interface `Store`, `Config`, `NewStore` |
| mail | `github.com/rolldone/donisgo/mail` | `Mailer`, `Mailable`, `SMTPConfig`, `NewMailerWithConfig` |
| auth | `github.com/rolldone/donisgo/auth` | `auth.Config` + `auth.New` (instans), utilitas JWT |
| secrets | `github.com/rolldone/donisgo/secrets` | `EncryptBlob` / `DecryptBlob` |
| uuid | `github.com/rolldone/donisgo/uuid` | `uuid.New()` generator UUIDv7 |
| console | `github.com/rolldone/donisgo/console` | Root CLI cobra + command `migrate` & `seed` (dapat diperluas) |
| plugins/plugin_registry | `github.com/rolldone/donisgo/plugins/plugin_registry` | Kontrak komunikasi antar-plugin tanpa circular import |

Tidak ada direktori `internal/`: seluruh paket di atas adalah API yang bisa
diimpor.

Wiring aplikasi
---------------
`app.Config` adalah entrypoint eksplisit dan injectable. Semua field opsional:
nil berarti "pakai default dari environment", jadi Anda hanya meng-override
yang perlu.

```go
package main

import (
    "log"

    "github.com/rolldone/donisgo/app"
    "github.com/rolldone/donisgo/plugins"
    myplugin "example.com/myapp/plugins/my_plugin"
)

func main() {
    application, err := app.New(app.Config{
        // Optional: inject handle yang sudah Anda miliki.
        // DB:    gormDB,      // nil → db.Open(db.ConfigFromEnv())
        // Store: myStore,     // nil → storage.NewStore(storage.LoadConfig())
        // KeyDB: keydbClient, // nil → keydb.New dari KEYDB_* env
        // Mail:  mailer,      // nil → mail.NewMailer()
        // CORS:  &cors.Config{},                 // nil → CORS_ALLOWED_ORIGINS
        // TrustedProxies: []string{"10.0.0.0/8"}, // nil → TRUSTED_PROXIES; empty → trust none
        Plugins: []plugins.Plugin{myplugin.New()},
    })
    if err != nil {
        log.Fatalf("failed to build app: %v", err)
    }
    log.Fatal(application.Run())
}
```

Catatan:

- `Plugins` non-nil → registry **segar dan terisolasi** milik `App` ini.
  `Plugins` nil → memakai registry default (perilaku transisional agar helper
  package-level yang deprecated tetap bekerja).
- `app.NewFromEnv()` adalah gula untuk `app.New(app.Config{})`.
- `app.Run(app.Options{...})` masih bekerja tetapi **deprecated**.
- donisgo **tidak** mendaftarkan plugin apa pun secara otomatis. Urutan plugin
  sepenuhnya milik pemanggil; pakai `Registry.EnsurePluginLast` untuk plugin
  yang memegang catch-all.
- Resolusi dependency tersedia via `GetStore()`, `GetMailer()`, `GetKeyDB()`,
  `GetRegistry()`.

Kontrak plugin
--------------
Implementasikan `plugins.Plugin` (`plugins/types.go`):

```go
type Plugin interface {
    ID() string
    RegisterServices(deps ServiceDeps) error
    RegisterMiddleware() []MiddlewareDescriptor
    RegisterRoutes(router *gin.Engine, admin *gin.RouterGroup, api *gin.RouterGroup) error
    Seed() error
    ConsoleCommands() []*cobra.Command
}
```

- `RegisterServices` menerima `ServiceDeps{DB *gorm.DB, Store storage.Store}`.
- `RegisterMiddleware` mengembalikan descriptor dengan `Target` (`global`,
  `admin`, `api`) dan `Priority` (lihat `plugins/middleware_priorities.go`).
- `RegisterRoutes` dipanggil setelah seluruh service teregistrasi.

Contoh plugin minimal:

```go
package myplugin

import (
    "github.com/gin-gonic/gin"
    "github.com/spf13/cobra"

    "github.com/rolldone/donisgo/plugins"
)

type Plugin struct{}

func New() plugins.Plugin { return &Plugin{} }

func (p *Plugin) ID() string { return "my-plugin" }

func (p *Plugin) RegisterServices(deps plugins.ServiceDeps) error { return nil }

func (p *Plugin) RegisterMiddleware() []plugins.MiddlewareDescriptor {
    return []plugins.MiddlewareDescriptor{{
        Name:     "my-plugin-logger",
        Target:   "admin",
        Priority: plugins.PriorityLogging,
        Handler:  func(c *gin.Context) { c.Next() },
    }}
}

func (p *Plugin) RegisterRoutes(_ *gin.Engine, admin, _ *gin.RouterGroup) error {
    admin.GET("/plugins/my-plugin/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "ok"})
    })
    return nil
}

func (p *Plugin) Seed() error                       { return nil }
func (p *Plugin) ConsoleCommands() []*cobra.Command { return nil }
```

Untuk membuat plugin baru dengan cepat, gunakan generator `plugin new` di repo
starter:

```bash
# di dalam proyek yang dibuat dari donisgo-template
go run ./cmd/console plugin new --id my-plugin --template minimal|crud|middleware
```

Plugin Registry (komunikasi antar-plugin)
-----------------------------------------
`plugins/plugin_registry` menyediakan berbagi interface antar-plugin tanpa
circular import:

1. Definisikan interface contract di `plugins/plugin_registry/registry.go`.
2. Plugin **PROVIDER** mendaftarkan implementasinya via `Register<X>Provider(...)`
   di `RegisterServices(deps)`.
3. Plugin **CONSUMER** memanggil delegation function.
4. Keduanya hanya mengimpor `plugin_registry`.
5. Setiap delegation function punya nil-guard: tanpa provider, kembalikan
   default aman (bukan panic).

Panduan menambah contract + contoh `EchoProvider` ada di komentar
`plugins/plugin_registry/registry.go`; error bersama di `errors.go`
(dideteksi via `errors.As`).

Registry & urutan plugin
------------------------
- `plugins.NewRegistry()` membuat registry terisolasi; wrapper package-level
  (`RegisterPlugins`, `EnsurePluginLast`, dst.) **deprecated** dan dipertahankan
  hingga v1.0.
- `EnsurePluginLast(p)` mendaftarkan `p` di akhir, memindahkan entri lama
  dengan ID sama (idempotent, dedup). Pakai untuk plugin yang memegang
  `router.NoRoute` — mis. plugin `pages` dari starter — agar route spesifik
  (api/admin/health) menang lebih dulu:

```go
reg := plugins.NewRegistry()
reg.RegisterPlugins([]plugins.Plugin{myplugin.New()})
reg.EnsurePluginLast(pagesplugin.New()) // catch-all selalu terakhir
```

Events
------
Bus event internal disediakan `events`. Handler dijalankan asinkron di goroutine
terpisah secara default. `Subscribe` mengembalikan fungsi unsubscribe;
`Publish` mengirim event.

```go
unsub := events.Subscribe("user.created", func(ctx context.Context, payload interface{}) {
   // tangani event
})
defer unsub()

events.Publish("user.created", map[string]interface{}{"id": "user123"})
```

Lihat `events/events_example_test.go` untuk contoh runnable.

Request–Reply (pola aman)
-------------------------
`events.RequestReply` membuat topik balasan unik per request, mem-publish
request dengan field `ReplyTo`, dan menunggu satu balasan dengan timeout.

- Topik balasan per-request (tanpa kanal balasan global) → tidak ada cross-talk.
- Pemanggil memberi timeout dan context untuk pembatalan.
- Subscriber membaca `ReplyTo` dari request dan mem-publish balasan ke topik itu.

Contoh (publisher):

```go
deadline := time.Now().Add(2 * time.Second).UnixNano()
req := map[string]interface{}{"id": "42", "Deadline": deadline}
resp, err := events.RequestReply(ctx, "user.query", req, 2*time.Second)
```

Contoh (subscriber):

```go
events.Subscribe("user.query", func(ctx context.Context, payload interface{}) {
   m, _ := payload.(map[string]interface{})
   if d, ok := m["Deadline"].(int64); ok && time.Now().UnixNano() > d {
      return // terlalu telat, lewati
   }
   replyTo, _ := m["ReplyTo"].(string)
   // kerjakan lalu publish hasil ke replyTo
   events.Publish(replyTo, map[string]interface{}{"ok": true})
})
```

Auth (JWT)
----------
donisgo menyediakan utilitas JWT tanpa layanan autentikasi atau middleware
bawaan. Implementasikan autentikasi sebagai plugin.

### Instans (disarankan)

`auth/config.go` menyediakan `auth.Config` + `auth.New(cfg) (*Auth, error)`.
Instance independen: dua instance dengan secret berbeda tidak dapat
memverifikasi token satu sama lain. Secret kosong → **error** (tidak ada
default insecure). TTL nol → default 10m/14d/1h.

```go
a, err := auth.New(auth.Config{
    AccessSecret:  os.Getenv("AUTH_ACCESS_SECRET"),
    RefreshSecret: os.Getenv("AUTH_REFRESH_SECRET"),
    ResetSecret:   os.Getenv("AUTH_RESET_SECRET"),
    // Opsional; nol → default (10m / 14d / 1h).
    AccessTTL:  15 * time.Minute,
    RefreshTTL: 14 * 24 * time.Hour,
    ResetTTL:   time.Hour,
})
if err != nil {
    log.Fatal(err)
}

token, err := a.SignAccessToken(userID)       // access token
refresh, err := a.SignRefreshToken(userID)    // refresh token
userID, err := a.ParseAccessToken(tokenStr)   // verifikasi access token
userID, err := a.ParseRefreshToken(tokenStr)  // verifikasi refresh token

accessExp := a.AccessExpirySeconds()   // mis. 900
refreshExp := a.RefreshExpirySeconds() // mis. 1209600
```

`auth.ConfigFromEnv()` memetakan variabel lingkungan lama (`AUTH_JWT_SECRET`
lebih dulu, lalu `JWT_ACCESS_SECRET` / `JWT_REFRESH_SECRET` / `JWT_RESET_SECRET`,
plus `JWT_*_EXP_SECONDS`) menjadi `Config` untuk `auth.New`.

### Deprecated: helper package-level

API package-level lama masih bekerja — ia meneruskan ke instance yang dibangun
dari environment saat startup — tetapi **deprecated** dan akan dihapus di v1.0:

```go
// Deprecated: gunakan instance *Auth.
token, err := auth.SignAccessToken(userID)
userID, err := auth.ParseAccessToken(tokenStr)
exp := auth.AccessExpirySeconds()
```

### Klaim kustom

```go
token, expTime, err := a.GenerateAccessTokenWithLevel(adminID, "admin", 15*time.Minute)
claims, err := a.ParseAccessTokenClaims(tokenStr) // claims.AdminID, claims.Level
```

### Opaque refresh token (non-JWT, untuk disimpan di DB)

```go
plainToken, hashedToken, err := auth.GenerateOpaqueRefreshToken() // 96 hex char acak
// simpan hashedToken di DB, kirim plainToken ke klien
receivedHash := auth.HashOpaqueToken(plainTokenFromClient) // bandingkan dengan DB
```

### Catatan implementasi

- Tanpa middleware auth / model user bawaan.
- Fungsi JWT **stateless dan agnostik DB**: hanya encode/decode data ke string
  token; query DB dilakukan terpisah memakai ID dari token.
- Nama kolom & struktur tabel sepenuhnya milik aplikasi.
- Rancang middleware sendiri untuk memvalidasi token dan menyuntikkan identitas
  ke `context.Context`.

### Rekomendasi keamanan

- Jangan commit secret; pakai secret manager.
- Access token pendek (`JWT_ACCESS_EXP_SECONDS` ~900), refresh token lebih
  panjang (`JWT_REFRESH_EXP_SECONDS` ~1209600).
- Selalu HTTPS; bila memakai cookie set `Secure`, `HttpOnly`, `SameSite`.
- Rotasi signing key; pertimbangkan `kid` bila memakai beberapa key.

### Testing

```go
func TestJWT(t *testing.T) {
    a, err := auth.New(auth.Config{
        AccessSecret:  "test-access-secret",
        RefreshSecret: "test-refresh-secret",
        ResetSecret:   "test-reset-secret",
    })
    if err != nil {
        t.Fatalf("auth.New: %v", err)
    }
    token, err := a.SignAccessToken("user123")
    if err != nil {
        t.Fatalf("sign: %v", err)
    }
    got, err := a.ParseAccessToken(token)
    if err != nil || got != "user123" {
        t.Fatalf("parse = (%q, %v)", got, err)
    }
}
```

DB (GORM)
---------
GORM adalah ORM utama (`db/`).

### Koneksi

- DSN dibangun dari `DB_TYPE`, `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`,
  `DB_NAME`, `DB_SSLMODE`.
- `db.Open(db.Config{...})` menerima handle eksplisit; `db.ConfigFromEnv()`
  memakai environment. Inject `*gorm.DB` ke service — jangan pakai variabel
  global.

### Pooling

- `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, `DB_CONN_MAX_LIFETIME_SEC`.

### Transaksi & context

- Teruskan `context.Context` dan, bila perlu, `*gorm.DB` transaksi dari handler
  ke service; gunakan `db.WithContext(ctx)`.
- Hindari transaksi panjang di jalur request user.

```go
err := db.WithTransaction(ctx, gdb, func(tx *gorm.DB) error {
    if err := svc.CreateItem(ctx, tx, req); err != nil {
        return err
    }
    return nil
})
```

`db/tx.go` menangani begin/commit/rollback termasuk panic.

### Migrasi

Command `migrate` ada di paket `console` (bukan di binary — donisgo tidak
mengirim binary). Aplikasi/starter memanggil `console.Execute()` dari
`main`-nya sendiri, sehingga subcommand `migrate` dan `seed` tersedia di CLI
aplikasi.

- File migrasi berada di `migrations/{db_type}` (core) dan
  `plugins/{plugin_id}/migrations/{db_type}` (plugin).
- Helper migrasi & integrasi CLI ada di `console/migrate.go`.
- `AutoMigrate` GORM untuk development; produksi pakai migrasi terstruktur.

Paket `console`
---------------
`console` mengekspos CLI framework:

```go
import (
    "github.com/rolldone/donisgo/console"
    "github.com/rolldone/donisgo/plugins"
    myplugin "example.com/myapp/plugins/my_plugin"
)

func main() {
    console.RegisterAdditionalPlugins([]plugins.Plugin{myplugin.New()})
    console.Execute()
}
```

- `NewRootCmd()` mengembalikan root cobra — tooling aplikasi (mis. generator
  `plugin new` milik starter) dapat menempelkan command sendiri.
- `RegisterAdditionalPlugins` mendaftarkan plugin ke registry default sekaligus
  memasang `ConsoleCommands()` mereka.
- `plugin new` **tidak** ada di library; ia tinggal di repo starter.

Variabel environment
--------------------
Library membaca environment untuk default `app.New`/`db.ConfigFromEnv`/
`storage.LoadConfig`/`keydb`/`mail`. Berkas `.env` sendiri urusan aplikasi
(donisgo tidak memuatnya); contoh lengkap ada di
`.env.example` repo starter.

**Database (GORM)**
- `DB_TYPE`=`postgres|mysql|mariadb`, `DB_HOST`, `DB_PORT`, `DB_NAME`,
  `DB_USER`, `DB_PASSWORD`, `DB_SSLMODE` (Postgres).
- `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, `DB_CONN_MAX_LIFETIME_SEC`.

**Auth / Security**
- `AUTH_JWT_SECRET` — secret kanonis (HS256). Legacy `JWT_SECRET`,
  `JWT_ACCESS_SECRET`, `JWT_REFRESH_SECRET` masih dibaca sebagai fallback.
- `JWT_ACCESS_EXP_SECONDS` (900), `JWT_REFRESH_EXP_SECONDS` (1209600),
  `JWT_RESET_EXP_SECONDS` (3600).

**Cache / Flash message (KeyDB/Redis)**
- `KEYDB_HOST`, `KEYDB_PORT`, `KEYDB_PASS`, `KEYDB_DB` — kosongkan host/port
  untuk menonaktifkan.

**Storage**
- `STORAGE_DRIVER`=`local|s3`, `STORAGE_ROOT` (local), `STORAGE_PUBLIC_URL`,
  `S3_BUCKET`, `S3_REGION`, `S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`.

**Mailer**
- `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS`, `SMTP_FROM`,
  `SMTP_USE_TLS`, `SMTP_STARTTLS`.

**HTTP**
- `CORS_ALLOWED_ORIGINS` — daftar origin dipisah koma.
- `TRUSTED_PROXIES` — lihat di bawah.

### Trusted proxies (`ClientIP`)

- `TRUSTED_PROXIES` — daftar IP/CIDR (dipisah koma) yang boleh mengeset
  `X-Forwarded-For` / `X-Real-IP`; gin memakainya untuk `ClientIP()`.
- **Kosong (default) = trust none**: header forwarded diabaikan dan
  `ClientIP()` mengembalikan socket peer. Ini default aman (gin default
  "percaya semua proxy" membuat IP bisa dipalsukan — issue #3).
- **Wajib diisi bila aplikasi berjalan di belakang reverse proxy**
  (nginx/caddy/traefik/NPM). Bila tidak, `ClientIP()` melaporkan IP proxy
  sehingga rate-limit berbasis IP dan audit log tidak berguna.
- Entri tidak valid ditolak dengan log `[WARNING]` dan seluruh daftar jatuh ke
  trust-none; startup tidak pernah gagal karena variabel ini.
- Override programatik: `app.Config.TrustedProxies` menang atas env (`nil` → env,
  explicit empty → trust none).

Bootstrapping (sudut pandang library)
-------------------------------------
Yang dilakukan `app.New`/`app.NewFromEnv`:

1. Membaca konfigurasi dari environment.
2. Membuka koneksi DB (GORM) bila tidak di-inject.
3. Menyiapkan storage (`local`/`s3`) bila tidak di-inject.
4. Menyiapkan KeyDB/Redis (opsional) dan mailer.
5. Menyiapkan router gin dengan CORS + trusted proxies.
6. Memakai registry plugin dari `Config.Plugins` (atau default bila nil).
7. `RegisterAllServices` → `AttachMiddleware` (prioritas) → `RegisterAllRoutes`.
8. (Opsional) route swagger bila diaktifkan.
9. `Run()` menyalakan HTTP server.

Tidak ada plugin yang ditambahkan otomatis — termasuk tidak ada serving
halaman/SPA.

Request flow
------------
1. Request masuk ke router gin milik `App`.
2. Middleware global berjalan menurut prioritas: gin default (Logger,
   Recovery) → CORS → middleware plugin.
3. Middleware auth (bila ada) adalah plugin, bukan bawaan.
4. Handler plugin/custom berjalan.
5. Handler memakai service plugin dengan dependency bersama (`*gorm.DB`,
   `storage.Store`, `keydb`, `mailer`).
6. Respons diserialisasi dan dikirim; hook plugin dapat memodifikasinya.

Migrasi dari `v0.1.x` (BREAKING)
--------------------------------
Perubahan penting di `v0.2.0`:

| Dihapus dari donisgo | Pengganti |
|---|---|
| `cmd/server`, `cmd/console` | `cmd/*` di `donisgo-template` |
| `plugins/pages` | `github.com/rolldone/donisgo-template/plugins/pages` — daftarkan manual + `EnsurePluginLast` |
| Generator `plugin new` (paket `console`) | `console/` di `donisgo-template` |
| `internal/pluginloader` | tidak ada — registrasi 100% via `app.Config.Plugins` |
| Docs app-level & `.env.example` | README + `.env.example` di `donisgo-template` |

**Aplikasi/sub-app yang selama ini mengandalkan serving halaman otomatis WAJIB
mendaftarkan `pages` eksplisit:**

```go
import pagesplugin "github.com/rolldone/donisgo-template/plugins/pages"

application, err := app.New(app.Config{
    Plugins: []plugins.Plugin{myplugin.New()},
})
// setelah semua plugin terdaftar, jamin pages terakhir:
application.GetRegistry().EnsurePluginLast(pagesplugin.New())
```

Detail lengkap ada di [`CHANGELOG.md`](CHANGELOG.md) entri `v0.2.0`.

Best practices
--------------
**Plugin-first:** implementasikan fitur (auth, manajemen user, logika bisnis)
sebagai plugin, bukan dengan mengubah core.

**Organisasi kode:** daftarkan route/middleware lewat plugin; jaga service layer
tetap lepas dari HTTP layer (terima `context.Context` + interface repository).

**Database:** teruskan `context.Context`; berikan `*gorm.DB` transaksi secara
eksplisit ke service; pakai `db.WithTransaction`.

**Testing:** mock DB & service; lihat `mail/mail_test.go` dan `app/app_test.go`
untuk pola.

**Keamanan:** jangan commit secret; validasi input di handler; batasi privilege
operasi yang dijalankan plugin.

Referensi file
--------------
- Bootstrap & wiring: `app/bootstrap.go`
- Kontrak & registry plugin: `plugins/types.go`, `plugins/registry.go`
- Prioritas middleware: `plugins/middleware_priorities.go`
- Komunikasi antar-plugin: `plugins/plugin_registry/`
- DB: `db/gorm.go`, transaksi `db/tx.go`, env `db/env.go`
- Auth: `auth/config.go`, `auth/jwt.go`, `auth/claims_tokens.go`
- Events: `events/` (`request_reply.go` untuk request–reply)
- Storage: `storage/config.go`
- Mailer: `mail/mailer.go`
- KeyDB: `keydb/client.go`
- CLI framework: `console/` (`root.go`, `migrate.go`, `seed.go`)
- Contoh konsumen library: `examples/library/`

Contoh konsumen lengkap ada di [`examples/library/`](examples/library) — modul
terpisah dengan `replace github.com/rolldone/donisgo => ../..`:

```bash
cd examples/library && go build ./...
```

Request flow diagram
--------------------
```
flowchart LR
   Client[Client HTTP] --> Server[app.Run / router]
   Server --> Router[Router]
   Router --> MWChain[Middleware Chain]

   subgraph Middlewares
      Core[Core middleware: logging, recovery, CORS]
      Auth[Auth middleware: plugin]
      PluginMW[Plugin middleware (priority-based)]
   end

   MWChain --> Core --> Auth --> PluginMW --> Handler[Handler]

   Handler --> Service[Service layer: plugin services]
   Service --> DB[GORM (db)]
   DB --> Service
   Service --> Handler

   Handler --> Response[Response -> Client]
   PluginMW --> Hooks[Plugin hooks / response modifiers]
   Hooks --> Response
```

Lisensi
-------
MIT — lihat [LICENSE](LICENSE).
