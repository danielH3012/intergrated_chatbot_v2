## 1. Overview

> Modul: **AI Assistant** | Feature Prefix: **`AIREG`**
> **Catatan struktur:** Prototype MVP — **berdiri sendiri**, belum terintegrasi ke Registration existing di [`Fixed-Asset/PRD/asset/`](../../../Fixed-Asset/PRD/asset/) (tidak pakai Asset Master, tidak pakai Approval). Tujuan prototype ini murni membuktikan alur: input dokumen/teks → ekstraksi AI → review → aset masuk ke daftar. Integrasi ke sistem produksi (Asset Master matching, Approval workflow) adalah keputusan terpisah setelah prototype ini tervalidasi — lihat `TAG-Samurai/AI-Assistant/riset-integrasi-ai.md` §5.2 untuk gambaran alur produksi jangka panjang.

### 1.1 Problem Statement

Registrasi aset manual (isi form field-by-field) lambat, terutama saat onboarding client baru yang sudah punya banyak aset tercatat di invoice/dokumen pembelian. User harus mengetik ulang informasi yang sebenarnya sudah ada di dokumen fisik/digital yang mereka punya.

### 1.2 Goals

- User bisa mendaftarkan aset dengan **upload dokumen** atau **deskripsi teks bebas** — digabung dalam **satu composer** (gaya chat), bukan cuma isi form manual
- AI mengekstrak & memetakan informasi ke field aset, tapi **tidak pernah membuat aset tanpa review manusia**
- AI **tidak boleh mengarang** data yang tidak ada di sumber — data yang ditemukan vs disimpulkan vs kosong harus bisa dibedakan user
- Aset yang dikonfirmasi **langsung masuk ke Asset List** — alur simpel, tanpa Asset Master, tanpa approval

### 1.3 Feature Summary

AI Registration adalah prototype berdiri sendiri: dari **Method Picker**, user pilih **Manual Form** atau **AI Assistant**. Kalau pilih AI Assistant, user masuk ke **AI Chat Input** — satu composer (mirip ChatGPT) tempat user bisa ketik deskripsi, attach dokumen (PDF/JPG/JPEG/PNG), atau dua-duanya sekaligus dalam satu pesan → AI mengekstrak informasi aset → hasil ditampilkan sebagai **Asset Draft** yang bisa dikoreksi/dilengkapi user → setelah dikonfirmasi, aset **langsung dibuat** dan muncul di **Asset List**. Tidak ada langkah pencocokan katalog (Asset Master) atau persetujuan (Approval) di MVP ini.

### 1.4 Scope

**In Scope (MVP):**
- **Method Picker** — 2 pilihan: Manual Form atau AI Assistant
- **AI Chat Input** — satu composer gabungan: teks bebas dan/atau attach dokumen (PDF, JPG, JPEG, PNG) dalam satu pesan, tidak terbatas ke invoice/foto saja selama formatnya memenuhi syarat
- Ekstraksi field: Nama Aset, Kategori, Merek, Model, Tanggal Pembelian, Harga Pembelian, Lokasi, jumlah unit (kalau terdeteksi >1 aset)
- Pemetaan hasil ekstraksi langsung ke field Asset (lihat [`04-data.md`](04-data.md) §4.3)
- Asset Draft — review, edit, lengkapi data oleh user sebelum konfirmasi
- Multi-asset detection — satu pesan berisi beberapa unit → beberapa baris draft, tampil sekaligus di tabel yang sama (bukan navigasi satu-satu)
- Guardrail: AI tidak mengarang; data hasil inferensi ditandai berbeda dari data yang ditemukan
- Confirm & Create → aset langsung tersimpan dan tampil di Asset List
- Manual Form — jalur non-AI, bisa dipilih langsung dari Method Picker atau sebagai fallback (F-AIREG-07)
- Asset List — Search, Filter, Sort, Pagination, Download, Column Visibility, row action Edit & Delete (single, tidak ada bulk) — lihat [`02-ui-design.md`](02-ui-design.md) §2.2a

**Out of Scope (MVP):**
- Registrasi via suara
- Pembuatan aset sepenuhnya otomatis (tanpa review manusia)
- AI agent percakapan kompleks (multi-turn negotiation soal data — composer ini satu kali kirim, bukan obrolan bolak-balik)
- Pencocokan/katalog Asset Master
- Approval workflow
- Pengayaan data dari sumber eksternal (mis. lookup harga pasar)
- Prediksi maintenance
- Penalaran multi-dokumen kompleks
- Workflow assignment/tagging otomatis
- Dashboard governance AI lengkap
- Foto aset — Asset List murni tabel data, tidak ada kolom/upload foto. Foto yang diupload sebagai input (kalau ada) cuma dipakai AI buat ekstraksi, tidak otomatis jadi foto aset

> Import massal dari file Excel/CSV (`Data Tools > Import`, lihat `riset-integrasi-ai.md` §5.3) adalah fitur **terpisah** — AI Registration ini fokus ke registrasi per-pesan/per-dokumen, bukan bulk spreadsheet.

### 1.5 Asumsi & Open Items Kunci

| # | Item | Asumsi/Rekomendasi | Status |
|---|------|---------------------|--------|
| 1 | Kategori & Lokasi | Prototype pakai **data mock** — bebas, tidak perlu nyambung ke Settings > Categories/Groups asli. Bisa free text atau dropdown dari daftar mock, terserah kebutuhan frontend | Resolved |
| 2 | Timing generate Asset ID | Di-generate saat Confirm & Create — aset langsung Active, tidak ada state Draft/Pending Approval yang menunda | Resolved |
| 3 | Prototype ini standalone | Belum terhubung ke mesin Registration produksi (`Fixed-Asset/PRD/asset/`) — integrasi jadi keputusan lanjutan | Open |
| 4 | Composer gabungan (teks + file dalam satu pesan) | AI Gateway menerima keduanya sekaligus kalau ada — ekstraksi menggabungkan konteks dari teks dan dokumen (bukan dua request terpisah) | Resolved |

---

## 2. User Flow

```
[+ Register Asset]
        ↓
   Pilih metode: [Manual Form] [AI Assistant]
        ↓ (AI Assistant)
┌───────────────────────────────────────────────────────────┐
│ AI CHAT INPUT (satu composer, gaya ChatGPT)                 │
│  Ketik deskripsi dan/atau attach dokumen (PDF/JPG/JPEG/PNG)  │
│  dalam satu pesan, lalu kirim                                │
└───────────────────────────────────────────────────────────┘
        ↓
┌───────────────────────────────────────────────────────────┐
│ AI MEMAHAMI → EKSTRAKSI → PEMETAAN FIELD                    │
│  AI baca teks + dokumen (kalau ada keduanya, digabung        │
│  jadi satu konteks), tentukan relevansi, ekstrak field       │
└───────────────────────────────────────────────────────────┘
        ↓
   ┌───────────────────────────┐
   │ Berapa aset terdeteksi?   │
   └───────────────────────────┘
     1 aset  → 1 baris di tabel Review
     >1 aset → N baris di tabel Review, semua tampil sekaligus di halaman yang sama
     0/tidak relevan → REJECT (lihat §8 edge case "Dokumen tidak relevan")
        ↓
┌───────────────────────────────────────────────────────────┐
│ REVIEW PENGGUNA (tabel, semua baris di halaman yang sama)   │
│  Tiap cell ditandai: Ditemukan / Perlu Dilengkapi /          │
│  (kalau ada) Disimpulkan AI — user edit/lengkapi per baris   │
└───────────────────────────────────────────────────────────┘
        ↓ Confirm & Create All
┌───────────────────────────────────────────────────────────┐
│ CREATE ASSET                                                 │
│  Validasi dasar per baris (required field, format) → baris   │
│  yang lengkap langsung dibuat, Asset ID di-generate           │
│  (AST-XXXXXX), muncul di Asset List. Baris belum lengkap      │
│  tetap di tabel, tidak ikut dibuat                            │
└───────────────────────────────────────────────────────────┘
```

**Alur Alternatif / Edge Cases:** lihat §8.

---

## 3. Halaman & Komponen UI

Ringkasan — detail lengkap di [`02-ui-design.md`](02-ui-design.md).

| Halaman/Komponen | Tipe | Fungsi |
|-------------------|------|--------|
| Method Picker | Dialog/step | Pilih Manual Form atau AI Assistant |
| AI Chat Input | Page/step | Composer gabungan: teks + attach file dalam satu pesan |
| Processing State | Inline/overlay | "AI sedang membaca..." |
| Asset Draft Review | Page/step | Tabel — satu baris per aset terdeteksi, semua tampil di halaman yang sama |
| Reject/Insufficient State | Inline message | Dokumen tidak relevan / kualitas buruk |
| Manual Form | Page | Jalur non-AI, field sama dengan Draft Review |
| Asset List | Page | Tabel hasil akhir — Search/Filter/Sort/Pagination/Download/Column Visibility; tidak ada kolom foto (lihat §1.4) |

---

## 4. Validasi & Error Messages

| Kondisi | Kapan Dicek | Pesan | Perilaku |
|---------|-------------|-------|----------|
| Attach file selain PDF/JPG/JPEG/PNG | Langsung saat file dipilih (client-side, sebelum upload jalan) | `"Unsupported file format. Please upload PDF, JPG, JPEG, or PNG."` | File ditolak, tidak masuk composer; chip file tidak muncul |
| Ukuran file > 10MB | Langsung saat file dipilih (client-side) | `"File too large. Max size: 10MB."` | File ditolak, tidak masuk composer |
| File kosong (0 bytes) | Langsung saat file dipilih (client-side) | `"This file appears to be empty. Please upload a valid document."` | File ditolak, tidak masuk composer |
| Klik Send tanpa teks maupun file terlampir | Saat klik `[Send]` | `"Please enter a description or attach a document."` | Tombol Send tetap disabled, tidak submit |
| Dokumen tidak relevan (§8) | Setelah AI Extraction selesai (server-side) | `"This document doesn't appear to contain asset information. Please try again or fill the form manually."` + CTA `[Fill Manually]` | Reject Banner, tidak masuk ke tabel Review |
| Dokumen kualitas buruk, AI tidak bisa baca sama sekali | Setelah AI Extraction selesai (server-side) | `"Unable to read this document clearly. Please try again or fill the form manually."` | Reject Banner, tidak masuk ke tabel Review |
| Required field kosong saat Confirm & Create All | Saat klik `[Confirm & Create All]` (server-side, per baris) | Inline per cell: `"[Field name] must not be empty"` | Baris terkait ditandai "Missing fields", tidak ikut dibuat — baris lain tetap diproses (partial success) |
| AI Gateway/provider error | Saat AI Extraction dipanggil (server-side) | `"AI is currently unavailable. Please fill the form manually."` | Fallback ke manual — lihat prinsip fallback `riset-integrasi-ai.md` §6.3 |
| Required field kosong saat Edit / Manual Form | Saat klik `[Save Changes]` / `[Create Asset]` | Inline per field: `"[Field name] must not be empty"` | Tidak tersimpan, tetap di form |

> Validasi format/ukuran file (3 baris pertama) **selalu client-side, di titik pemilihan file** — jadi user tahu masalahnya sebelum sempat klik Send, tidak buang-buang panggilan ke AI Gateway untuk file yang jelas-jelas tidak valid.

---

## 5. States & Feedback

| State | Kondisi | Tampilan |
|-------|---------|---------|
| Idle (Composer) | Belum ada teks maupun file | Composer kosong, `[Send]` disabled |
| File Rejected (client-side) | File dipilih tapi format salah / >10MB / 0 bytes | Error inline di bawah composer, file tidak masuk |
| File Attached | File berhasil dilampirkan | Chip file muncul di composer, bisa dihapus, teks tetap bisa diketik bersamaan |
| Processing | AI sedang ekstraksi | Loading indicator + teks progres ("Membaca...", "Mengekstrak informasi...") |
| Draft Ready | Ekstraksi selesai, ≥1 draft valid | Asset Draft Review ditampilkan |
| Partial/Low Confidence | Dokumen kualitas buruk tapi masih ada info terbaca | Draft ditampilkan dengan banyak field kosong/flagged, banner peringatan |
| Rejected | Dokumen tidak relevan / tidak terbaca sama sekali | Pesan reject + CTA fallback manual |
| Multi-Row | >1 aset terdeteksi | Semua baris tampil sekaligus di tabel yang sama, tiap baris punya Row Status independen |
| Toast sukses Create | Confirm & Create All berhasil (semua baris) | `"[N] asset(s) registered successfully."` |
| Toast partial Create | Sebagian baris berhasil, sebagian masih Missing fields | `"[N] asset(s) created. [M] row(s) still need required fields."` |

---

## 6. Logging

| Log | Isi |
|-----|-----|
| AI Registration Session Log | Timestamp, user, apakah pesan berisi teks/file/keduanya, nama file (jika ada), jumlah draft dihasilkan, jumlah dikonfirmasi vs dibatalkan — untuk audit & evaluasi akurasi AI |
| Asset Create Log | Dicatat saat Confirm & Create/Manual Form berhasil — Asset ID, timestamp, sumber (`ai_registration` / `manual`) |
| Asset Edit/Delete Log | Dicatat saat Edit atau Delete berhasil — Asset ID, field yang berubah (Edit) atau nama aset (Delete), timestamp, user |

---

## 7. User Stories

### 7.1 Epic

Sebagai **user**, saya ingin **mendaftarkan aset lewat satu composer AI (teks dan/atau dokumen)**, sehingga **saya tidak perlu mengetik ulang informasi yang sudah ada di invoice/dokumen pembelian**.

### 7.2 Story Breakdown

| ID | User Story | Prioritas |
|----|-----------|:--------:|
| **AIREG-01** | Ketik deskripsi dan/atau attach dokumen dalam satu composer untuk registrasi aset | P0 |
| **AIREG-02** | Lihat tabel Asset Draft hasil ekstraksi AI dengan cell bersumber jelas (ditemukan/disimpulkan/kosong) | P0 |
| **AIREG-03** | Edit/lengkapi cell yang salah atau kosong, per baris, langsung di tabel | P0 |
| **AIREG-04** | Konfirmasi (Confirm & Create All) → baris yang lengkap langsung dibuat dan tampil di Asset List | P0 |
| **AIREG-05** | Deteksi multi-aset dalam satu pesan, semua baris tampil sekaligus di tabel yang sama | P1 |
| **AIREG-06** | Fallback ke form manual saat dokumen ditolak/AI gagal | P0 |
| **AIREG-07** | Lihat, cari, filter, urutkan, dan download daftar aset di Asset List | P0 |
| **AIREG-08** | Edit aset yang sudah tersimpan langsung dari Asset List | P0 |
| **AIREG-09** | Hapus satu aset dari Asset List dengan konfirmasi | P0 |

### 7.3 Acceptance Criteria

**[AIREG-01] — AI Chat Input**

| Given | When | Then |
|-------|------|------|
| User di AI Chat Input | Ketik teks saja, klik `[Send]` | Diproses sebagai input teks |
| User di AI Chat Input | Attach file saja (tanpa teks), klik `[Send]` | Diproses sebagai input dokumen |
| User di AI Chat Input | Ketik teks DAN attach file, klik `[Send]` | Diproses sebagai satu pesan gabungan — AI pakai teks + dokumen sebagai satu konteks |
| Composer kosong (tidak ada teks maupun file) | — | `[Send]` disabled |

**[AIREG-02/03] — Review**

| Given | When | Then |
|-------|------|------|
| Ekstraksi selesai, semua field penting ditemukan | — | Tabel Draft tampil, tiap baris lengkap dengan cell bersumber "Ditemukan" |
| Field tidak ada di sumber (mis. Merek tidak disebut) | — | Cell dikosongkan dengan indikator "Perlu Dilengkapi" — **AI tidak mengarang nilai** |
| User edit cell yang salah | Ubah nilai di satu baris | Nilai baru tersimpan di baris itu saja, baris lain tidak berubah |

**[AIREG-04] — Create**

| Given | When | Then |
|-------|------|------|
| Satu/lebih baris punya semua field wajib terisi (lengkap dari AI atau dilengkapi user) | Klik `[Confirm & Create All]` | Baris "Ready" dibuat jadi aset, Asset ID digenerate, tampil di Asset List; baris lain (kalau masih "Missing fields") tetap di tabel |
| Semua baris "Missing fields" | Klik `[Confirm & Create All]` | Tidak ada yang dibuat, error inline di cell yang kosong pada tiap baris |

**[AIREG-05] — Multi-Asset**

| Given | When | Then |
|-------|------|------|
| Dokumen/teks menyebut 3 unit laptop identik/berbeda | Ekstraksi selesai | 3 baris tampil sekaligus di tabel yang sama (bukan navigasi satu-satu) |
| User edit Location di baris 1, baris 2 & 3 sudah lengkap | Klik `[Confirm & Create All]` | Ketiga baris (setelah baris 1 dilengkapi) dibuat jadi aset dalam satu aksi |

**[AIREG-06] — Fallback**

| Given | When | Then |
|-------|------|------|
| Dokumen/teks tidak relevan (bukan info aset) | — | Pesan reject + `[Fill Manually]` membawa ke form manual kosong |
| AI Gateway error/timeout | — | Pesan unavailable + fallback ke form manual |

**[AIREG-07] — Asset List**

| Given | When | Then |
|-------|------|------|
| Ada beberapa aset tersimpan | Buka Asset List | Tabel tampil, urut Created At descending (terbaru di atas) |
| User ketik di Search | Cari by Name/Asset ID/Category/Brand/Model/Location | Tabel terfilter sesuai kata kunci |
| User pilih Filter Category/Brand/Location | Terapkan filter | Tabel terfilter sesuai kombinasi filter |
| User klik Download | Pilih CSV/XLSX | File `Assets.csv`/`Assets.xlsx` terunduh sesuai data yang sedang tampil |
| Belum ada aset sama sekali | Buka Asset List | Empty state `"No assets yet"` + CTA `[+ Register Asset]` |
| Search/filter tidak match apa pun | — | Empty state `"No results found"` |

**[AIREG-08] — Edit**

| Given | When | Then |
|-------|------|------|
| User klik row action `[Edit]` | — | Form terbuka, pre-filled dari data aset saat ini; Asset ID read-only |
| User ubah field, klik `[Save Changes]` | Field wajib (Name/Category/Location) tetap terisi | Aset terupdate, toast `"Asset updated successfully."`, kembali ke Asset List |
| User kosongkan field wajib, klik `[Save Changes]` | — | Error inline per field, tidak tersimpan |

**[AIREG-09] — Delete**

| Given | When | Then |
|-------|------|------|
| User klik row action `[Delete]` | — | Dialog konfirmasi `"Delete [Asset Name]? This action cannot be undone."` |
| User klik `[Delete]` di dialog | — | Aset terhapus, toast `"Asset deleted successfully."`, baris hilang dari tabel |
| User klik `[Cancel]` di dialog | — | Tidak ada perubahan, dialog tertutup |

---

## 8. Edge Cases

| # | Skenario | Perilaku yang Diharapkan |
|---|----------|---------------------------|
| 1 | Dokumen/teks lengkap, semua field penting tersedia | Asset Draft lengkap dibuat, semua field "Ditemukan" |
| 2 | Data tidak ada di sumber (mis. ada nama aset tapi tanpa brand) | Brand dikosongkan dengan flag "Perlu Dilengkapi" — **AI tidak boleh mengarang** |
| 3 | Banyak aset dalam satu pesan | AI kenali jumlah unit, tampilkan sebagai beberapa baris di tabel Review yang sama (lihat AIREG-05) |
| 4 | Dokumen kualitas buruk (tulisan tak terbaca) | Diproses best-effort; banner "Kualitas dokumen kurang jelas, mohon periksa kembali" + field low-confidence ditandai; atau minta kirim ulang / lanjut manual |
| 5 | Hanya foto aset tanpa info tertulis relevan | Draft parsial dibuat (nama/kategori dari visual bila memungkinkan), sisanya "Perlu Dilengkapi" |
| 6 | Dokumen/teks tidak relevan (mis. foto menu makanan) | Ditolak — pesan reject + CTA `[Fill Manually]` (lihat AIREG-06) |
| 7 | Input teks tidak lengkap (mis. "laptop Dell 5 unit buat tim desain") | Ambil info yang ada (nama, merek, jumlah), field lain "Perlu Dilengkapi" |
| 8 | Harga/tanggal/vendor/warranty tidak ada di sumber | Field dikosongkan — **dilarang keras** diisi nilai karangan AI (guardrail §9 non-negotiable) |
| 9 | User hapus semua baris di tabel Review (0 baris tersisa) | Tombol `[Confirm & Create All]` disabled; tampilkan opsi kembali ke Input atau batalkan sesi sepenuhnya |
| 10 | User kirim teks DAN file sekaligus dalam satu pesan | AI gabungkan keduanya jadi satu konteks ekstraksi (bukan diproses dua kali terpisah) — lihat §1.5 #4 |

---

## 9. Batasan & Guardrail AI (non-negotiable)

- AI **tidak boleh mengarang** harga, tanggal, vendor, warranty, atau keberadaan aset yang tidak ada di sumber
- Data hasil **inferensi** AI (disimpulkan, bukan ditemukan eksplisit) harus **bisa dibedakan** secara visual dari data yang **ditemukan** langsung di sumber (lihat §2.3 di `02-ui-design.md` untuk treatment visual)
- Final asset creation **tetap melalui validasi dasar** (required field, format) — AI tidak melewati validasi apa pun

---

## Changelog

| Tanggal | Perubahan | Oleh |
|---------|-----------|------|
| 2026-08-28 | Initial version — AI Registration MVP: input dokumen/teks, ekstraksi, review, create asset | PRD Builder |
| 2026-08-28 | Simplifikasi: hapus Serial Number, hapus role/permission matrix, hapus Asset Master & Approval — prototype jadi standalone (input → asset list langsung) | PRD Builder |
| 2026-08-28 | Multi-asset: semua baris draft tampil sekaligus di tabel yang sama (bukan navigasi satu-satu), Confirm & Create All dengan partial success per baris | PRD Builder |
| 2026-08-28 | Tambah Manual Form, Asset List table tools (search/filter/sort/pagination/download/column visibility), Edit & Delete (single row). Hapus "Save as Draft" (gak ada halaman resume) — F-AIREG di-renumber jadi 01-12. Resolve TODO batas file (10MB), max karakter teks (2000) | PRD Builder |
| 2026-08-28 | Audit konsistensi: fix key JSON `asset_name`→`name` (04-data.md), tambah endpoint discard draft row (row action 🗑 wajib panggil backend, bukan cuma state FE), tambah edge case "semua baris dihapus" | PRD Builder |
| 2026-08-28 | Method Picker jadi 2 pilihan (Manual Form / AI Assistant). Upload Document & Describe in Text digabung jadi satu **AI Chat Input** (composer gaya ChatGPT — teks dan/atau attach file dalam satu pesan). F-AIREG & AIREG story di-renumber (F-AIREG-01 s.d. 11, AIREG-01 s.d. 09) | PRD Builder |
