## 5. Dependencies & Constraints

### 5.1 Dependencies

| Modul/Fitur | Hubungan |
|-------------|----------|
| `AI-Assistant/riset-integrasi-ai.md` §6 (Base AI Platform) | AI Registration **bergantung** pada AI Gateway (provider abstraction) — **belum ada dokumentasi terpisah**, base platform ini masih perlu ditulis (lihat Open Item §5.3 di bawah) |
| `_foundation/error-handling-convention.md` | Response contract, null rendering, atomic write |
| `_foundation/data-contract-validation.md` | Validation Rules Table format |

> Prototype ini **tidak bergantung** ke `Fixed-Asset/PRD/asset/` maupun `settings-asset-master/` — berdiri sendiri dengan entitas Asset sederhana sendiri (lihat `04-data.md` §4.1). Integrasi ke sistem produksi (Asset Master, Draft/Approval) adalah keputusan terpisah di masa depan.

### 5.2 Prerequisites

- AI Gateway/Base Platform harus tersedia (provider terkonfigurasi) — kalau belum, seluruh fitur ini fallback ke manual (F-AIREG-07)

### 5.3 Constraints

- Dokumen sumber yang diupload **tidak otomatis** jadi lampiran permanen aset — perlu keputusan terpisah `[TODO: konfirmasi PM]`
- Base AI Platform (Gateway, provider config, security layer) **belum punya dokumentasi PRD sendiri** — modul ini ditulis dengan asumsi Gateway tersedia sebagai black box; begitu Base Platform PRD ditulis, bagian ini perlu direview ulang
- Prototype ini **tidak punya** state Draft/Pending Approval — Confirm & Create selalu langsung menghasilkan aset final

### 5.4 Open Items (rekap dari 01-overview.md §1.5)

| # | Item | Rekomendasi Sementara |
|---|------|------------------------|
| 1 | Kategori & Lokasi — pakai data mock atau nyambung ke data Settings asli | **Resolved** — pakai data mock, bebas |
| 2 | Batas ukuran file upload | **Resolved** — max 10MB, min: tidak boleh kosong (0 bytes); nilai default prototype |
| 3 | Dokumen sumber jadi lampiran permanen atau tidak | Belum diputuskan |
| 4 | Kapan/apakah prototype ini diintegrasikan ke sistem produksi (Asset Master, Approval) | Keputusan lanjutan setelah prototype tervalidasi |

---

## Changelog

| Tanggal | Perubahan | Oleh |
|---------|-----------|------|
| 2026-08-28 | Initial version — dependencies ke asset/settings-asset-master, constraints, rekap open items | PRD Builder |
| 2026-08-28 | Simplifikasi: lepas dependency ke `Fixed-Asset/PRD/asset/` dan `settings-asset-master/` — prototype berdiri sendiri, tanpa Asset Master dan tanpa Approval | PRD Builder |
