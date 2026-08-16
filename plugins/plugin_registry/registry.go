// Package plugin_registry menyediakan mekanisme komunikasi antar plugin
// melalui interface contract.
//
// Masalah yang dipecahkan: dua plugin yang saling butuh layanan akan jatuh ke
// circular import jika mereka saling import langsung. Solusinya — keduanya
// import package ini saja:
//
//	┌─────────────┐  register  ┌──────────────────┐   delegate   ┌─────────────┐
//	│ Plugin A    │───────────▶│  plugin_registry │◀─────────────│ Plugin B    │
//	│ (PROVIDER)  │  Register  │  (interface +    │   panggil    │ (CONSUMER)  │
//	└─────────────┘  XProvider│   nil-guard)     │  delegation  └─────────────┘
//
// Cara kerja:
//  1. Definisikan interface contract di package ini (mis. ChatProvider).
//  2. Plugin PROVIDER mendaftarkan implementasinya via Register<X>Provider(...)
//     pada RegisterServices(deps).
//  3. Plugin CONSUMER memanggil delegation function (mis. SendMessage(...)).
//  4. Provider & consumer TIDAK saling import — keduanya hanya import
//     plugin_registry → tidak ada circular import.
//  5. Setiap delegation function WAJIB punya nil-guard: kalau provider belum
//     terdaftar, kembalikan default yang aman (bukan panic).
package plugin_registry

import "context"

// ══════════════════════════════════════════════════════════════════════════
// CARA MENAMBAH CONTRACT BARU
// ══════════════════════════════════════════════════════════════════════════
//
// Salin pola di bawah (pakai EchoProvider sebagai contoh), lalu sesuaikan:
//
//	// 1. Definisikan interface yang dibutuhkan consumer.
//	type NamaProvider interface {
//		Method(ctx context.Context, ...) (..., error)
//	}
//
//	// 2. Variabel package-level untuk menyimpan implementasi terdaftar.
//	var namaProvider NamaProvider
//
//	// 3. Fungsi register — dipanggil oleh plugin PROVIDER di RegisterServices.
//	func RegisterNamaProvider(p NamaProvider) { namaProvider = p }
//
//	// 4. Fungsi delegation — dipanggil oleh plugin CONSUMER.
//	func NamaDelegation(ctx context.Context, ...) (..., error) {
//		if namaProvider == nil {
//			return <nil-safe default>, nil
//		}
//		return namaProvider.Method(ctx, ...)
//	}
//
// Aturan:
//   - Interface & tipe data yang dibagikan (struct/error) didefinisikan DI SINI,
//     bukan di plugin, supaya tidak ada dependency antar plugin.
//   - Nama delegation function diawali kata kerja (Get*, Send*, Validate*, dst).
//   - Nil-guard wajib: tanpa provider, sistem harus tetap berfungsi (no-op aman).
//   - Jangan simpan state bisnis di sini — ini hanya kontrak & dispatcher.
//
// ══════════════════════════════════════════════════════════════════════════

// ──────────────────────────────────────────────────────────────────────────
// CONTOH CONTRACT (pola baku — ganti dengan contract nyata sesuai kebutuhan)
// ──────────────────────────────────────────────────────────────────────────

// EchoProvider adalah CONTOH interface contract. Ia mendemonstrasikan pola
// provider/delegation. Saat plugin bisnis pertama dibuat, ganti/blok contoh
// ini dengan contract yang relevan (mis. ChatProvider, AuthProvider, dll).
type EchoProvider interface {
	// Echo mengembalikan pesan yang sama (placeholder demonstrasi pola).
	Echo(ctx context.Context, msg string) (string, error)
}

var echoProvider EchoProvider

// RegisterEchoProvider mendaftarkan implementasi EchoProvider. Dipanggil oleh
// plugin provider di RegisterServices.
func RegisterEchoProvider(p EchoProvider) {
	echoProvider = p
}

// Echo memanggil provider terdaftar. Nil-guard: jika belum ada provider yang
// mendaftar, kembalikan pesan apa adanya (no-op aman).
func Echo(ctx context.Context, msg string) (string, error) {
	if echoProvider == nil {
		return msg, nil
	}
	return echoProvider.Echo(ctx, msg)
}
