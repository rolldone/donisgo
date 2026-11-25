# SPEC-005 — donisgo Pure Library: pemisahan starter & tooling ke `donisgo-template`

- Status    : **DRAFT** — menunggu persetujuan pemilik (Open Questions §7)
- Tanggal   : 2026-11-25
- Pengusul  : Manager (donis-agent) atas rencana pemilik
- Target    : donisgo `v0.2.0` (BREAKING, jalur v0.x) + lahirnya
              `donisgo-template v0.1.0`
- Ref       : SPEC-004 (library-ization, `v0.1.0`), issue
              [#17](https://github.com/rolldone/donisgo/issues/17), topic
              Historic `1795594944725` (penuntasan SPEC-004)

## 1. Latar Belakang

donisgo lahir sebagai satu repo framework: library + binary (`cmd/server`,
`cmd/console`) + scaffold proyek (`plugin new`) + plugin konten
(`plugins/pages`) + dokumentasi aplikasi. SPEC-004 (`v0.1.0`) mengubah donisgo
menjadi library yang bisa diimpor; sejak itu seluruh paket inti publik dan
`internal/` hanya `pluginloader`.

Rencana pemilik (2026-11-25): **donisgo menjadi pure library**. Semua yang
bukan API library dipindah ke repo baru
**`github.com/rolldone/donisgo-template`** (sejajar/sibling). Ekosistem
menjadi dua peran yang tegas:

| Repo | Peran | Cara konsumsi |
|---|---|---|
| `donisgo` | Pure library (API inti) | di-pin via `go.mod` (`go get ...@vX.Y.Z`) |
| `donisgo-template` | Starter aplikasi + tooling | di-clone/fork (GitHub *Use this template*), bukan di-pin |

## 2. Tujuan

1. donisgo: nol binary, nol konten konkret, nol scaffolding aplikasi — hanya
   paket yang bisa diimpor & dikomposisikan.
2. donisgo-template: satu sumber starter resmi (server, CLI, scaffold, plugin
   konten, docs aplikasi) yang mengonsumsi donisgo sebagai dependensi pinned.
3. Batas tanggung jawab & siklus rilis kedua repo tegas dan terdokumentasi.

## 3. Non-Goals

- Tidak mengubah perilaku/logika paket yang berpindah — pindah murni
  (penyesuaian import path) + penyesuaian titik registrasi.
- Tidak mengubah API paket yang tetap di donisgo.
- Tidak membangun plugin marketplace/registry publik.

## 4. Keputusan Desain (D1–D6)

### D1 — Repo baru bernama `donisgo-template`

- Alasan nama: "framework" = donisgo itu sendiri (ambigu); repo baru berisi
  starter + tooling → **template** akurat dan cocok dengan fitur GitHub
  *Use this template*. Alternatif (donisgo-starter/base) tidak dipakai.
- Visibilitas **public** (mengikuti donisgo). Skeleton awal dibuat Manager
  2026-11-25 (README, .gitignore; branch `master`, commit `a41c336`);
  konten awal akan ditimpa/terisi saat fase 2–5.

### D2 — Boundary pemindahan

| Artefak | Keputusan | Alasan |
|---|---|---|
| `plugins/pages` | **PINDAH** | Plugin konten konkret (StaticFS serving). Konten ≠ kontrak. |
| `cmd/server`, `cmd/console` | **PINDAH** | Pure library = nol binary. |
| Command `plugin new` + skeleton yang dihasilkannya (sub_app/webapp, migrasi) | **PINDAH** | Meng-generate wujud aplikasi = ranah starter, bukan library. |
| Docs app-level README (quickstart, env, runbook, deploy) | **PINDAH** | README donisgo tinggal API/install/versi. |
| `.env.example` bagian app-specific | **PINDAH** | Milik starter. |
| 10 paket publik (`app`, `auth`, `db`, `events`, `keydb`, `mail`, `secrets`, `storage`, `uuid`, kontrak `plugins`) | **TETAP** | Inti library. |
| `plugins/plugin_registry` | **TETAP** | Mekanisme komunikasi antar-plugin = infrastruktur, bukan konten. |
| Paket `console` (framework) + command generic `migrate`, `seed` | **TETAP** | Tetap bisa diimpor sub-app (janji v0.1.2). Implementasi `plugin new` yang pindah, bukan framework-nya. |
| `internal/pluginloader` | **TETAP (menyesuaikan)** | Berhenti register `pages`; evaluasi lanjut di fase 4 (jika tidak ada tanggung jawab tersisa → gabung ke `app`). |
| `examples/library`, `docs/specs` | **TETAP** | Milik library. |

### D3 — Penyesuaian pasca-pemindahan (di donisgo)

- `internal/pluginloader`: hapus registrasi core `pages` (`RegisterCorePlugins*`);
  `RegisterLastPlugins*` dievaluasi — jika hanya melayani pages → hapus,
  jika masih dipakai user-plugin → pertahankan.
- `console/root.go`: daftar command tanpa `plugin new`; tetap host `migrate`,
  `seed`, dan extensibility untuk starter.
- `app/bootstrap.go`: komentar yang menyebut `plugins/pages` diperbarui.
- README dipecah: API-level tetap di donisgo; quickstart/env/deploy pindah.
- CHANGELOG `v0.2.0` (BREAKING): catat penghapusan + path baru di template repo.

### D4 — Struktur & module path repo baru

- Module: `github.com/rolldone/donisgo-template`; layout mengikuti donisgo:
  `cmd/server`, `cmd/console`, `plugins/pages`, console starter (`plugin new`),
  docs aplikasi — supaya diff pemindahan minimal.
- Template mengonsumsi donisgo via `go.mod` pinned; **tanpa** direktif
  `replace` ke path lokal di commit (`replace` hanya untuk dev lokal).
- `plugin new` menghasilkan modul sub-app yang mengimpor
  `github.com/rolldone/donisgo/...` (pemisahan `frameworkModulePath` hasil
  fix #13 tetap berlaku).

### D5 — Versi & migrasi

- donisgo: **clean cut di `v0.2.0`** (BREAKING) — tanpa masa deprecated.
  Alasan: jalur `v0.x` membolehkan breaking pada bump minor; konsumen
  `plugins/pages`/cmd di luar modul sendiri diketahui nol; ekosistem muda.
  Migration note wajib di CHANGELOG + README ekosistem.
- donisgo-template: tag pertama **`v0.1.0`** (repo baru, riwayat rilis
  sendiri), paruh waktu dengan donisgo `v0.2.0`.

### D6 — Urutan eksekusi (5 fase)

| Fase | Isi | Risiko |
|---|---|---|
| 1 | Lahirkan repo `donisgo-template` (skeleton + push) | Rendah — **SELESAI 2026-11-25** |
| 2 | Pindahkan `plugins/pages` + `cmd/server` + `cmd/console` (fresh copy, catat commit asal di CHANGELOG template) + gates kedua repo | Sedang |
| 3 | Pindahkan scaffold `plugin new` + docs app-level + `.env.example`; bersihkan `console` donisgo | Sedang |
| 4 | Penyesuaian `pluginloader`/`console/root`/`app` + evaluasi pluginloader; gates penuh + scaffold end-to-end di salinan temp | Sedang |
| 5 | Rilis: donisgo `v0.2.0` (tag + CHANGELOG) & template `v0.1.0`; README ekosistem dua arah | Rendah |

Gerbang kualitas tiap fase (kedua repo): `gofmt -l .` kosong, `go vet ./...`,
`go build ./...`, `go test ./...`, commit pesan eksplisit.

## 5. Risiko & Mitigasi

| Risiko | Mitigasi |
|---|---|
| Konsumen eksternal `plugins/pages`/cmd terputus | Migration note `v0.2.0` + README ekosistem menunjuk path baru; breaking pada bump minor adalah jalur resmi `v0.x` |
| Drift antara scaffold dan starter | Scaffold & skeleton hidup di repo yang sama (template) — satu sumber kebenaran |
| `pluginloader` jadi kosong/mubazir | Fase 4 evaluasi: gabung ke `app` bila tidak ada tanggung jawab tersisa |

## 6. Acceptance Criteria (SPEC ini)

- [ ] donisgo: `go build ./...` hijau tanpa `cmd/` & `plugins/pages`; nol
      binary; README API-level; CHANGELOG `v0.2.0` memuat BREAKING + catatan
      migrasi.
- [ ] donisgo-template: gates hijau; `cmd/server` & `cmd/console` jalan;
      `plugin new` end-to-end menghasilkan sub-app yang bisa di-build dengan
      donisgo versi pinned.
- [ ] README kedua repo saling merujuk (peta ekosistem).
- [ ] Tag: donisgo `v0.2.0`, donisgo-template `v0.1.0`.

## 7. Open Questions (butuh jawaban pemilik)

- **OQ-1** — Skeleton awal repo template (README + .gitignore, commit
  `a41c336`) disetujui sebagai titik mulai? (rekomendasi: ya — akan terisi
  saat fase 2–5)
- **OQ-2** — LICENSE: donisgo saat ini **tidak punya** file LICENSE. Tambahkan
  (mis. MIT) di kedua repo sebelum ekosistem makin dipakai? (rekomendasi: ya)
- **OQ-3** — `plugin new` dipindah utuh dulu, refactor menyusul jika perlu?
  (rekomendasi: ya — pindah dulu, jangan campur refactor)
- **OQ-4** — Layout template mengikuti donisgo (`plugins/pages`, `cmd/*` di
  path yang sama)? (rekomendasi: ya — diff pemindahan minimal)
