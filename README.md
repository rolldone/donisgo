# donisgo

[![Go Reference](https://pkg.go.dev/badge/github.com/rolldone/donisgo.svg)](https://pkg.go.dev/github.com/rolldone/donisgo)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A composable Go web-framework **library** — gin + GORM + a plugin architecture —
built to be imported, not to be run. Donisgo ships the infrastructure (app
wiring, database, migrations, plugin contracts, event bus, cache, storage, mail,
JWT) and leaves the application shape to you.

It contains **no binaries, no inner `internal/` packages, and no scaffolding**:
every package listed below is importable API.

## Install & pin

```bash
go get github.com/rolldone/donisgo@v0.2.0
```

Always pin an explicit version in `go.mod`; see [Versioning & releases](#versioning--releases).

## Minimal usage

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
        // Every field is optional: nil means "use the env-driven default".
        Plugins: []plugins.Plugin{myplugin.New()},
    })
    if err != nil {
        log.Fatalf("failed to build app: %v", err)
    }
    log.Fatal(application.Run())
}
```

`app.NewFromEnv()` is shorthand for `app.New(app.Config{})`. Resolved
dependencies are available through `GetStore()`, `GetMailer()`, `GetKeyDB()` and
`GetRegistry()`.

## Package map

| Package | Import path | What you get |
|---|---|---|
| `app` | `github.com/rolldone/donisgo/app` | `Config`, `New`, `NewFromEnv`, `(*App).Run` and dependency accessors |
| `plugins` | `github.com/rolldone/donisgo/plugins` | `Plugin` contract, `ServiceDeps`, `MiddlewareDescriptor`, `Registry`, priority constants |
| `plugins/plugin_registry` | `github.com/rolldone/donisgo/plugins/plugin_registry` | Cross-plugin interface contracts with nil-guarded delegation |
| `db` | `github.com/rolldone/donisgo/db` | `Open(db.Config{...})`, `WithTransaction` |
| `auth` | `github.com/rolldone/donisgo/auth` | `Config` + `New` instance API, JWT sign/verify, opaque refresh tokens |
| `events` | `github.com/rolldone/donisgo/events` | `NewBus()` publish/subscribe + request/reply |
| `keydb` | `github.com/rolldone/donisgo/keydb` | `New(cfg)` → `*Client` (flash messages, raw Redis) |
| `storage` | `github.com/rolldone/donisgo/storage` | `Store` interface, `Config`, `NewStore` (local / s3) |
| `mail` | `github.com/rolldone/donisgo/mail` | `Mailer`, `Mailable`, `SMTPConfig` |
| `secrets` | `github.com/rolldone/donisgo/secrets` | `EncryptBlob` / `DecryptBlob` |
| `uuid` | `github.com/rolldone/donisgo/uuid` | `New()` UUIDv7 generator |
| `console` | `github.com/rolldone/donisgo/console` | Cobra root command with `migrate` and `seed`, extensible by callers |

Full API reference: [pkg.go.dev/github.com/rolldone/donisgo](https://pkg.go.dev/github.com/rolldone/donisgo).

## Plugins

A plugin implements `plugins.Plugin`; the registry is instance-scoped, so each
application (or test) owns its own registrations:

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
        Handler:  func(c *gin.Context) { c.Next() },
        Priority: plugins.PriorityLogging,
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

Donisgo **registers no plugins of its own**. Ordering belongs to the caller;
plugins that own a catch-all route (for example a SPA/pages plugin) must be
registered last:

```go
reg := plugins.NewRegistry()
reg.RegisterPlugins([]plugins.Plugin{myplugin.New()})
reg.EnsurePluginLast(catchAllPlugin) // idempotent, de-duplicated by ID
```

## Auth

`auth` provides token mechanics only — no authentication service, middleware or
user model:

```go
a, err := auth.New(auth.Config{
    AccessSecret:  os.Getenv("AUTH_ACCESS_SECRET"),
    RefreshSecret: os.Getenv("AUTH_REFRESH_SECRET"),
    ResetSecret:   os.Getenv("AUTH_RESET_SECRET"),
})
if err != nil {
    log.Fatal(err) // a missing secret is an error — never an insecure default
}

token, err := a.SignAccessToken(userID)
sub, err := a.ParseAccessToken(token)
```

Instances are isolated: two `*auth.Auth` values with different secrets cannot
verify each other's tokens. `auth.ConfigFromEnv()` maps the legacy environment
variables (`AUTH_JWT_SECRET` first, then `JWT_ACCESS_SECRET` /
`JWT_REFRESH_SECRET` / `JWT_RESET_SECRET`) into a `Config`. The package-level
helpers are deprecated wrappers kept until v1.0.

## Database

```go
gdb, err := db.Open(db.Config{Driver: "postgres", DSN: os.Getenv("DATABASE_URL")})
if err != nil {
    log.Fatal(err)
}

err = db.WithTransaction(ctx, gdb, func(tx *gorm.DB) error {
    return svc.CreateItem(ctx, tx, req)
})
```

Prefer injecting `*gorm.DB` into services over package globals. `db.ConfigFromEnv()`
and `db.GetGormDB()` remain as deprecated environment bridges.

## Ecosystem

| Repository | Role | How to consume |
|---|---|---|
| **donisgo** (this repo) | Pure library — importable core API | `go get github.com/rolldone/donisgo@vX.Y.Z` |
| [donisgo-template](https://github.com/rolldone/donisgo-template) | Starter application + tooling — `cmd/server`, `cmd/console`, the `plugin new` generator, the `pages` content plugin, frontend skeleton, app docs and `.env.example` | clone / fork / GitHub *Use this template* |

Since **v0.2.0 serving pages/SPA is no longer automatic**. Applications that need
HTML/SPA serving must register the `pages` plugin explicitly from the starter and
keep it last:

```go
import pagesplugin "github.com/rolldone/donisgo-template/plugins/pages"

application, err := app.New(app.Config{Plugins: []plugins.Plugin{myplugin.New()}})
// ...
application.GetRegistry().EnsurePluginLast(pagesplugin.New())
```

See the [donisgo-template README](https://github.com/rolldone/donisgo-template)
for quickstart, environment variables, migrations, deployment and the plugin
generator.

## Versioning & releases

Donisgo follows [SemVer](https://semver.org/) with Git tags. Releases are
documented in [CHANGELOG.md](CHANGELOG.md) ([Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format).

⚠️ Donisgo is on the `v0.x` track: backward-incompatible changes may appear on a
**minor** bump (`v0.1.x` → `v0.2.0`). Read the changelog before upgrading.
`v0.2.0` is a breaking release — see the migration notes in the changelog.

## Examples

A complete buildable consumer lives in [`examples/library/`](examples/library) —
its own module with `replace github.com/rolldone/donisgo => ../..`:

```bash
cd examples/library && go build ./...
```

## License

MIT — see [LICENSE](LICENSE).
