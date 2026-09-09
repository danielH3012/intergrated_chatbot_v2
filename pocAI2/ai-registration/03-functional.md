## 3. Functional Requirements

> Ditujukan untuk: **Engineering (FE & BE)**
> Prototype berdiri sendiri — Confirm & Create membuat record Asset baru langsung (tanpa Asset Master, tanpa Approval), bukan memanggil mesin Registration produksi di `Fixed-Asset/PRD/asset/`.

> **Acuan foundation (jangan tulis ulang aturannya — referensikan):**
> - [`error-handling-convention.md`](../../../_foundation/error-handling-convention.md) (C-2) — HTTP→UI, null rendering, atomic write
> - [`data-contract-validation.md`](../../../_foundation/data-contract-validation.md) (M-2/M-3) — API Response Contract + Validation Rules Table
> - `riset-integrasi-ai.md` §6.3 (MCP/Tool-Calling)

> **Perubahan terakhir:** Upload Document (F-AIREG-01 lama) dan Text Input (F-AIREG-02 lama) digabung jadi satu **Combined Input** — satu composer, teks dan/atau file dalam satu pesan. "Save as Draft" tetap dihapus dari MVP (gak ada halaman resume). F-AIREG di-renumber jadi 01-11.

### 3.1 Ringkasan Fungsi

| ID | Fungsi | Deskripsi | Prioritas |
|----|--------|-----------|:--------:|
| F-AIREG-01 | Combined Input (Chat Composer) | Terima teks bebas dan/atau file (PDF/JPG/JPEG/PNG) dalam satu pesan, validasi format & ukuran | P0 |
| F-AIREG-02 | AI Extraction | Panggil AI Gateway, ekstrak field dari teks+file dengan guardrail anti-hallucination | P0 |
| F-AIREG-03 | Multi-Asset Detection | Deteksi >1 unit/aset dalam satu input, tampilkan sebagai beberapa baris di tabel yang sama | P1 |
| F-AIREG-04 | Draft Review & Edit | User edit/lengkapi cell per baris di tabel Asset Draft | P0 |
| F-AIREG-05 | Confirm & Create All | Buat Asset untuk semua baris yang lolos validasi, sekali aksi | P0 |
| F-AIREG-06 | Reject Handling | Tangani dokumen/teks tidak relevan/tidak terbaca | P0 |
| F-AIREG-07 | Fallback to Manual | Alihkan ke form manual saat AI gagal/ditolak | P0 |
| F-AIREG-08 | Manual Form Submit | Buat Asset langsung dari form manual (tanpa AI) | P0 |
| F-AIREG-09 | Asset List — Table Behavior | Search, filter, sort, pagination, download, column visibility | P0 |
| F-AIREG-10 | Edit Asset | Ubah data aset yang sudah tersimpan, row action dari Asset List | P0 |
| F-AIREG-11 | Delete Asset | Hapus satu aset dari Asset List, dengan konfirmasi | P0 |

### 3.2 Business Logic per Fungsi

**[F-AIREG-01] Combined Input (Chat Composer)**
- Satu composer: textarea (teks bebas, tanpa batas minimum, max **2000 karakter**) + tombol Attach (file: PDF/JPG/JPEG/PNG)
- User bisa kirim: **teks saja**, **file saja**, atau **teks + file sekaligus** dalam satu pesan — `[Send]` disabled kalau dua-duanya kosong
- Validasi file — max size **10MB**, min: file tidak boleh kosong (0 bytes); nilai default prototype, bisa disesuaikan kalau ada alasan bisnis spesifik nanti
- **Validasi format & ukuran file dicek client-side, langsung saat file di-attach** (sebelum kirim/API call apa pun) — file yang gagal validasi ditolak di titik itu juga, tidak pernah sampai terkirim ke AI Gateway
- File yang lolos validasi disimpan sementara (session-scoped) sampai draft dikonfirmasi/dibatalkan

**[F-AIREG-02] AI Extraction**
- Input (teks, file, atau keduanya) dikirim **sebagai satu payload** ke AI Gateway (bukan langsung ke provider — `riset-integrasi-ai.md` §3.1) dengan system prompt yang secara eksplisit melarang AI mengarang nilai untuk field yang tidak ditemukan
- Kalau teks dan file dikirim bersamaan, AI **menggabungkan keduanya jadi satu konteks ekstraksi** — bukan dua proses/dua hasil terpisah (`01-overview.md` §1.5 #4)
- Output terstruktur (structured output/JSON) berisi per field: `{ value, source: "extracted" | "inferred" | "empty", confidence }`
- **Guardrail non-negotiable** (`01-overview.md` §9): field tanpa dasar di sumber **wajib** `source: "empty"`, tidak boleh diisi. Ini dicek di **level prompt/schema**, bukan cuma UI
- Input tidak relevan atau tidak terbaca sama sekali → AI Gateway mengembalikan status reject, bukan draft kosong (lihat F-AIREG-06)

**[F-AIREG-03] Multi-Asset Detection**
- AI mendeteksi indikasi jumlah unit (mis. "5 unit", baris berulang di dokumen) → hasilkan N baris draft dalam **satu sesi**, masing-masing field diekstrak independen per baris
- Semua baris ditampilkan **sekaligus di halaman Review yang sama** sebagai tabel (`02-ui-design.md` §2.2) — bukan navigasi satu-per-satu
- Tiap baris punya Row Status independen (`Ready` / `Missing fields`) berdasarkan kelengkapan field wajibnya masing-masing

**[F-AIREG-04] Draft Review & Edit**
- Semua cell pada tiap baris **dapat diedit** oleh user, terlepas dari source-nya (Ditemukan/Disimpulkan/Kosong)
- Mengubah satu cell hanya memengaruhi baris itu — baris lain tidak berubah
- User bisa menghapus satu baris (row action `[🗑]`) kalau AI salah mendeteksinya sebagai unit terpisah — **wajib** memanggil `DELETE /ai-registration/drafts/{id}` (`04-data.md` §4.4), bukan cuma dihapus dari state frontend, karena Confirm & Create All memproses semua draft `pending` di sesi itu

**[F-AIREG-05] Confirm & Create All**
- Backend validasi dasar **per baris**: required field (`name`, `category`, `location`), format (`01-overview.md` §4)
- Baris yang lolos validasi (`Ready`) **langsung dibuat** jadi Asset (tidak ada state Draft/Pending Approval) — Asset ID (`AST-XXXXXX`) di-generate per baris, langsung muncul di Asset List
- Baris yang gagal validasi (`Missing fields`) **tidak ikut dibuat** dan tetap tersisa di tabel — kegagalan satu baris tidak menggagalkan baris lain (partial success, konsisten pola di `02-ui-design.md` §2.4)
- Ini **bukan** panggilan ke mesin Registration produksi (`Fixed-Asset/PRD/asset/`) — prototype punya endpoint create sendiri (lihat `04-data.md` §4.4)
- Tidak ada "Save as Draft" — kalau user meninggalkan halaman Review tanpa Confirm, sesi/draft yang belum dikonfirmasi dianggap ditinggalkan (tidak ada jaminan resume)

**[F-AIREG-06] Reject Handling**
- Kondisi reject: input tidak mengandung informasi aset yang bisa diidentifikasi (mis. teks/dokumen tidak berkaitan), atau kualitas terlalu buruk untuk dibaca sama sekali
- AI Gateway mengembalikan response terstruktur dengan `status: "rejected"` + alasan singkat — FE menampilkan Reject Banner (`02-ui-design.md` §2.2), bukan draft kosong
- Reject **berbeda** dari "kualitas rendah tapi masih terbaca sebagian" — kasus terakhir tetap menghasilkan draft parsial dengan banner peringatan (bukan reject)

**[F-AIREG-07] Fallback to Manual**
- Trigger: AI Gateway timeout/error (lihat `riset-integrasi-ai.md` §6.3 "Fallback ke UI manual"), atau user klik `[Fill Manually]` dari Reject Banner
- Membawa user ke form manual sederhana (§2.2 di `02-ui-design.md`, form kosong), **tidak ada field yang di-carry-over** dari percobaan AI yang gagal

**[F-AIREG-08] Manual Form Submit**
- Field sama dengan kolom di Draft Review: Asset Name, Category, Brand, Model/Type, Purchase Date, Purchase Price, Location
- Validasi & hasil **sama persis** dengan F-AIREG-05 (required: `name`, `category`, `location`; Asset ID auto-generate `AST-XXXXXX`) — beda titik masuk saja (form kosong vs hasil ekstraksi AI), bukan logic create yang berbeda
- Bisa diakses langsung dari Method Picker (`[📝 Manual Form]`) atau sebagai fallback dari F-AIREG-07

**[F-AIREG-09] Asset List — Table Behavior**
- Search: tidak case-sensitive; by Name, Asset ID, Category, Brand, Model/Type, Location
- Filter: Category, Brand, Location (multi-select); Purchase Date (range)
- Sort: default Created At descending; kolom lain sortable
- Pagination: default 10; pilihan 10/25/50/100/All
- Download: CSV/XLSX dari data yang sedang tampil (setelah search/filter diterapkan), filename `Assets.csv`/`Assets.xlsx`
- Column Visibility: semua kolom bisa di-hide kecuali Asset ID dan Name (`02-ui-design.md` §2.2a)

**[F-AIREG-10] Edit Asset**
- Row action `[Edit]` di Asset List → form pre-filled dari data aset saat ini, field sama dengan Manual Form (Asset Name, Category, Brand, Model/Type, Purchase Date, Purchase Price, Location)
- **Asset ID tidak bisa diedit** (read-only, immutable — konsisten `04-data.md` §4.1)
- Validasi sama dengan create: required `name`, `category`, `location`
- Berhasil → toast `"Asset updated successfully."`, kembali ke Asset List dengan data terbaru

**[F-AIREG-11] Delete Asset**
- Row action `[Delete]` di Asset List → dialog konfirmasi (`02-ui-design.md` §2.2), bukan langsung terhapus
- Konfirmasi → hard delete record Asset, baris hilang dari tabel
- Berhasil → toast `"Asset deleted successfully."`
- **Single row saja** — tidak ada bulk select/bulk delete di MVP ini

### 3.3 Validation Rules Table

| Field | Rule | Ditegakkan Saat |
|-------|------|-------------------|
| Teks/file di composer | Wajib salah satu terisi | Sebelum tombol "Send" aktif |
| name | Required, ≤120 char | Confirm & Create All / Manual Form Submit / Edit |
| category | Required | Confirm & Create All / Manual Form Submit / Edit |
| location | Required | Confirm & Create All / Manual Form Submit / Edit |
