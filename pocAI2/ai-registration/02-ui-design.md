## 2. UI Design Requirements

> Ditujukan untuk: **Frontend / Prototype builder**
> Copy/string UI: **Bahasa Inggris** (konsisten `prd-conventions.md`)
> Prototype berdiri sendiri — tidak terikat ke halaman Registration existing di Fixed-Asset. Entry point cukup tombol `[+ Register Asset]` yang mengarah ke Asset List sederhana.

---

### 2.1 Pages & Components

| Halaman/Komponen | Tipe | Keterangan |
|--------------------|------|-----------|
| Method Picker | Modal/Dialog | Muncul saat klik `[+ Register Asset]` — pilih Manual Form atau AI Assistant |
| AI Chat Input | Step/Page | Satu composer gaya chat — teks bebas dan/atau attach file (PDF/JPG/JPEG/PNG) dalam satu pesan |
| Processing | Inline overlay | Loading state saat AI ekstraksi |
| Asset Draft Review | Step/Page | Tabel — satu baris per aset terdeteksi, semua tampil di halaman yang sama (1 aset = 1 baris, N aset = N baris) |
| Reject/Fallback Banner | Inline component | Saat dokumen ditolak atau AI gagal |
| Manual Form | Page | Form kosong — jalur non-AI (F-AIREG-08) |
| Asset List | Page | Tabel hasil akhir — lengkap dengan Search/Filter/Sort/Pagination/Download/Column Visibility |

---

### 2.2 Layout & Wireframe Spec

```
Modal: Method Picker
├── Title: "Register Asset"
├── Subtitle: "How do you want to add this asset?"
├── Option Card: [📝 Manual Form]   — "Fill the form yourself"
├── Option Card: [✨ AI Assistant]  — "Describe it or attach a document — AI fills the form for you"
└── [Cancel]
```

```
Page: AI Chat Input (gaya ChatGPT — satu composer, teks + file digabung)
├── Header: [← Cancel]  |  "Register Asset" | Step indicator: [1. Input] → [2. Review] → [3. Done]
│     [← Cancel] membawa balik ke Method Picker — input teks/file yang belum dikirim dibuang
├── Message area (empty state)
│   └── "Describe the asset, or attach a document — AI will extract the details for you."
├── Composer (sticky di bawah)
│   ├── Attached file chip (kondisional, kalau user sudah attach)
│   │     [📄 invoice-laptop.pdf]  [x Remove]
│   ├── Textarea (auto-grow)
│   │     Placeholder: "e.g. 5 Dell Latitude 5450 laptops, purchased for the design team..."
│   │     Max 2000 karakter, char counter di pojok
│   ├── [📎 Attach] — buka file picker (PDF/JPG/JPEG/PNG, max 10MB)
│   └── [➤ Send] — disabled kalau teks kosong DAN tidak ada file terlampir
└── Boleh kirim: teks saja / file saja / teks + file sekaligus dalam satu pesan
```

```
Overlay: Processing
├── Spinner/animasi
├── Teks progresif (ganti tiap ~2 detik, kosmetik):
│   "Reading your message..." → "Extracting asset information..." → "Mapping to asset fields..."
└── Tidak ada tombol cancel — proses ekstraksi singkat (hitungan detik), cancel tidak diperlukan di MVP ini
```

```
Page: Asset Draft Review
├── Header: "Register Asset" | Step indicator: [1. Input] → [2. Review] → [3. Done]
├── Subtitle: "[N] asset(s) detected — review and edit before creating"
├── Source Banner (kondisional)
│   ├── Kualitas dokumen rendah: ⚠ "Document quality is low — please review extracted data carefully."
│   └── Input teks: ℹ "Fields not mentioned in your description are left empty for you to fill."
├── Table — satu baris per aset terdeteksi, semua baris tampil sekaligus di halaman yang sama
│   ├── Columns: [🗑] | Asset Name * | Category * | Brand | Model/Type | Purchase Date | Purchase Price | Location * | Row Status
│   ├── Tiap cell: value inline-editable (klik → jadi input), treatment visual per source (lihat §2.3)
│   ├── Row Status: ✓ "Ready" (semua required terisi) / ⚠ "Missing fields" (required masih kosong — baris ini tidak ikut Create sampai dilengkapi)
│   ├── Row action [🗑]: hapus baris ini dari draft (mis. AI salah deteksi sebagai unit terpisah)
│   └── Contoh 3 baris:
│         Row 1: Dell Latitude 5450 | IT Equipment > Laptop | Dell | Latitude 5450 | 20 Aug 2026 | Rp18,500,000 | (kosong ⚠) | ⚠ Missing fields
│         Row 2: Dell Latitude 5450 | IT Equipment > Laptop | Dell | Latitude 5450 | 20 Aug 2026 | Rp18,500,000 | Warehouse A   | ✓ Ready
│         Row 3: Dell Latitude 5450 | IT Equipment > Laptop | Dell | Latitude 5450 | 20 Aug 2026 | Rp18,500,000 | Warehouse A   | ✓ Ready
├── Preview thumbnail dokumen sumber (collapsible, buat cross-check manual — hanya tampil kalau input berupa dokumen, tidak ada untuk input teks)
└── Navigation: [Back]  [Confirm & Create All →]
```

> Tidak ada "Save as Draft" — kalau user klik `[Back]` atau meninggalkan halaman, sesi/draft yang belum di-Confirm dianggap ditinggalkan (tidak ada jaminan bisa dilanjutkan nanti).

> **Confirm & Create All** membuat aset untuk **semua baris berstatus "Ready"** sekaligus. Baris berstatus "Missing fields" **tidak ikut dibuat** dan tetap tersisa di halaman (bukan gagal total) — user lengkapi lalu klik Confirm & Create All lagi, atau hapus barisnya kalau memang tidak relevan.

```
Reject/Fallback Banner (inline, menggantikan Review saat gagal)
├── Icon ⚠
├── Judul: "We couldn't process this document"
├── Body: pesan sesuai kondisi (lihat 01-overview.md §4)
└── Actions: [Try Another File]  [Fill Manually →]
```

```
Page: Manual Form (dipilih langsung dari Method Picker, atau fallback dari F-AIREG-07 — submit-nya F-AIREG-08)
├── Header: "Register Asset"
├── Form (kosong, tidak ada carry-over dari percobaan AI yang gagal):
│   ├── Asset Name *
│   ├── Category *
│   ├── Brand
│   ├── Model/Type
│   ├── Purchase Date
│   ├── Purchase Price
│   └── Location *
└── Navigation: [Back]  [Create Asset →]
```

> Validasi & hasil Manual Form **sama persis** dengan Confirm & Create All (required: Asset Name, Category, Location; Asset ID auto-generate format `AST-XXXXXX`) — cuma beda titik masuknya (form manual vs hasil ekstraksi AI).

```
Page: Asset List
├── Header: "Assets" | Button: [+ Register Asset]
├── TableTools Toolbar [right-aligned]
│   ├── Icon: Search → "Search by name, asset ID, category, brand..."
│   ├── Icon: Filter → reveal Filter Panel
│   │   ├── Dropdown: "Category" [multi-select]
│   │   ├── Dropdown: "Brand" [multi-select]
│   │   ├── Dropdown: "Location" [multi-select]
│   │   └── Date Range: "Purchase Date" — From / To
│   ├── Icon: Download → overlay "Download as CSV" / "Download as XLSX"
│   └── Icon: Column Visibility (⋯)
├── Table
│   ├── Columns: Asset ID | Name | Category | Brand | Model/Type | Purchase Date | Purchase Price | Location | Created At | Actions
│   ├── Row Actions: [Edit] [Delete]
│   └── Sort default: Created At descending (aset terbaru di atas — beda dari konvensi Name-ascending modul lain, karena tujuan utama halaman ini melihat hasil registrasi terbaru)
├── Pagination: default 10; pilihan 10 / 25 / 50 / 100 / All
└── Empty state: "No assets yet" + CTA [+ Register Asset]
```

```
Page: Edit Asset (row action [Edit])
├── Header: "Edit Asset"
├── Form (pre-filled dari data existing, field sama dengan Manual Form):
│   ├── Asset Name *
│   ├── Category *
│   ├── Brand
│   ├── Model/Type
│   ├── Purchase Date
│   ├── Purchase Price
│   └── Location *
│   └── Asset ID: "AST-000042" (read-only, tidak bisa diedit — immutable)
└── Navigation: [Cancel]  [Save Changes →]
```

```
Dialog: Delete Asset (row action [Delete])
├── Title: "Delete Asset"
├── Body: "Delete [Asset Name]? This action cannot be undone."
└── Actions: [Cancel]  [Delete]
```

> **Tidak ada kolom/upload foto aset di prototype ini.** Kalau dokumen yang diupload berupa foto, foto itu cuma dipakai AI buat ekstraksi teks (mis. baca merek/model dari foto), **tidak** otomatis jadi `photo_url` aset — karena foto yang diupload bisa jadi foto dokumen pembelian, bukan foto asetnya sendiri. Bisa ditambahkan di fase berikutnya kalau dibutuhkan.

---

### 2.2a Global Table Rules — Asset List

| Rule | Detail |
|------|--------|
| Toolbar | Right-aligned: Search \| Filter \| Download \| Column Visibility (⋯) |
| Pagination | Default 10; pilihan 10/25/50/100/All |
| Search | Tidak case-sensitive; by Name, Asset ID, Category, Brand, Model/Type, Location |
| Filter | Category, Brand, Location (multi-select), Purchase Date (range) |
| Sort | Default: Created At descending (terbaru di atas) |
| Download | CSV/XLSX; filename `Assets.csv` |
| Column Visibility — tidak bisa di-hide | Asset ID, Name |
| Row Action | `[Edit]` — buka form pre-filled, field sama dengan Manual Form. `[Delete]` — dialog konfirmasi, single row saja (tidak ada bulk delete di MVP ini) |

---

### 2.3 Cell Source Indicator — Visual Treatment

> Guardrail wajib (§9 di `01-overview.md`): data yang **ditemukan** vs **disimpulkan** vs **kosong** harus bisa dibedakan user secara visual, per cell di tabel. Pakai badge palette existing (5 warna: biru/hijau/kuning/merah/hitam — `prd-conventions.md`).

| Source | Kapan dipakai | Visual Treatment (per cell) | Indikator |
|--------|----------------|-------------------------------|-----------|
| **Ditemukan** (Extracted) | Nilai eksplisit ada di dokumen/teks sumber | Cell normal, border solid tipis | titik hijau kecil di pojok cell |
| **Disimpulkan** (Inferred) | AI menyimpulkan dari konteks (mis. kategori dari nama produk, bukan disebutkan eksplisit) | Cell dengan background tint kuning muda, border dashed | titik kuning + tooltip `"AI-suggested — please verify"` |
| **Perlu Dilengkapi** (Empty, required) | Field wajib tapi tidak ditemukan sama sekali | Cell kosong, border merah, placeholder "Required" | titik merah, ikut menentukan Row Status "Missing fields" |
| **Kosong** (Empty, optional) | Field opsional tidak ditemukan | Cell kosong netral, placeholder `—` (konsisten null rendering C-2 §8) | tanpa indikator |

> **Non-negotiable:** AI tidak pernah mengisi cell dengan nilai karangan untuk terlihat "lengkap". Field yang tidak ditemukan **selalu** ditampilkan kosong (Missing/Kosong), tidak pernah diisi placeholder value yang terlihat seperti data asli.

---

### 2.4 States

| State | Kondisi | Tampilan |
|-------|---------|---------|
| Empty (Composer) | Belum ada teks maupun file | Composer kosong dengan placeholder, `[Send]` disabled |
| File Attached | File berhasil di-attach, teks boleh kosong/diisi | Chip file muncul di composer, `[Send]` aktif |
| Processing | AI sedang ekstraksi | Overlay spinner + progressive text |
| Draft Ready | Ekstraksi selesai | Asset Draft Review — tabel terisi sesuai source, 1 baris per aset terdeteksi |
| Low Confidence | Dokumen kualitas buruk | Banner peringatan + banyak cell Missing/low-confidence |
| Rejected | Dokumen tidak relevan/tidak terbaca | Reject Banner + CTA fallback |
| AI Unavailable | Gateway/provider error | Pesan unavailable + fallback ke manual form |
| Multi-Row | >1 aset terdeteksi | Semua baris tampil sekaligus di tabel yang sama, masing-masing Row Status independen |
| Confirming | Klik Confirm & Create All, request berjalan | Tombol loading state, baris "Ready" disabled sementara diproses |
| Partial Success | Sebagian baris berhasil dibuat, sebagian masih "Missing fields" | Toast `"[N] asset(s) created. [M] row(s) still need required fields."` — baris yang berhasil hilang dari tabel/ditandai selesai, baris belum lengkap tetap tampil |
| Success | Semua baris berhasil dibuat | Toast `"[N] asset(s) registered successfully."` + redirect ke Asset List |
| Validation Error | Required field kosong pada satu/lebih baris saat Confirm & Create All | Baris terkait ditandai "Missing fields" + border merah pada cell yang kosong, baris lain tetap diproses |
| Asset List — Empty | Belum ada aset sama sekali | Illustration + `"No assets yet"` + CTA `[+ Register Asset]` |
| Asset List — Empty (filtered) | Search/filter tidak match apa pun | Illustration + `"No results found"` (tanpa CTA) |
| Asset List — Loading | Fetch data | Skeleton rows |
| Edit Asset — Toast sukses | Save Changes berhasil | `"Asset updated successfully."` |
| Delete Asset — Toast sukses | Delete dikonfirmasi | `"Asset deleted successfully."` |

---

### 2.5 Responsive

| Screen | Desktop | Mobile |
|--------|---------|--------|
| Method Picker | 3 card sejajar | 1 card per baris, stacked |
| AI Chat Input | Composer centered, max-width 640px, sticky di bawah | Full width, composer sticky di bawah viewport |
| Asset Draft Review | Tabel penuh, semua kolom terlihat + preview dokumen di samping | Tabel scroll horizontal, atau 1 card per baris (stacked) sebagai alternatif; preview dokumen collapsible di atas |
| Manual Form | Form 1 kolom, max-width 640px | Full width |
| Asset List | Tabel penuh + toolbar sejajar kanan | Toolbar jadi icon-only, tabel scroll horizontal |
