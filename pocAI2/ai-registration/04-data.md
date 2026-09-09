## 4. Data Requirements

> Ditujukan untuk: **Programmer (Backend & Frontend)**
> Prototype berdiri sendiri — entitas Asset di sini **bukan** entitas Asset produksi di [`Fixed-Asset/PRD/asset/04-data.md`](../../../Fixed-Asset/PRD/asset/04-data.md), sekadar cukup untuk membuktikan alur input → ekstraksi → review → daftar aset. Format Asset ID (`AST-XXXXXX`) tetap disamakan dengan konvensi produksi.

### 4.1 Entitas

#### AI Registration Session (tabel `ai_registration_session`)

| Field | Type | Required | Deskripsi |
|-------|------|:--------:|-----------|
| id | UUID | ✓ | Primary key |
| user_id | UUID | ✓ | User yang memulai sesi |
| source_file_url | String | ✗ | URL dokumen yang di-attach — nullable, **minimal salah satu dari `source_file_url`/`raw_text` harus terisi** (composer gabungan, `01-overview.md` §1.5 #4) |
| raw_text | String | ✗ | Teks dari composer — nullable, lihat catatan di atas |
| status | Enum | ✓ | `processing` · `draft_ready` · `rejected` · `error` |
| reject_reason | String | ✗ | Alasan singkat jika `status = rejected` (mis. `"irrelevant_document"`, `"unreadable"`) |
| created_at | Timestamp | ✓ | — |

#### AI Registration Draft (tabel `ai_registration_draft`)

| Field | Type | Required | Deskripsi |
|-------|------|:--------:|-----------|
| id | UUID | ✓ | Primary key |
| session_id | UUID | ✓ | FK → `ai_registration_session` |
| sequence_no | Integer | ✓ | Urutan draft dalam sesi (multi-asset — mulai dari 1) |
| fields | JSON | ✓ | Lihat struktur §4.2 di bawah |
| status | Enum | ✓ | `pending` · `confirmed` · `discarded` |
| created_asset_id | UUID | ✗ | FK → `Asset.id` — terisi setelah Confirm & Create berhasil |
| low_confidence | Boolean | ✓ | `true` jika dokumen sumber kualitas rendah (untuk banner peringatan) |
| created_at | Timestamp | ✓ | — |
| updated_at | Timestamp | ✓ | Terakhir diedit user |

#### Asset (sederhana, khusus prototype ini)

| Field | Type | Required | Deskripsi |
|-------|------|:--------:|-----------|
| id | UUID | ✓ | Primary key |
| asset_id | String | ✓ | Alias-code, format `AST-XXXXXX` — prefix tetap `AST`, nomor urut 6 digit mulai `000001`, auto-increment, unik, immutable |
| name | String | ✓ | Nama aset (free text dari hasil ekstraksi/input user) |
| category | String | ✓ | Kategori aset — **data mock**, bebas (free text atau dropdown dari daftar mock, mis. "IT Equipment > Laptop", "Furniture", "Vehicle") |
| brand | String | ✗ | Merek |
| model_type | String | ✗ | Model/Type |
| purchase_date | Date | ✗ | Tanggal pembelian |
| purchase_price | Decimal | ✗ | Harga pembelian |
| location | String | ✓ | Lokasi aset — **data mock**, bebas (mis. "Warehouse A", "Office 2nd Floor") |
| created_at | Timestamp | ✓ | — |

### 4.2 Struktur `fields` (JSON)

Setiap key **persis sama** dengan nama field di entitas Asset (§4.1: `name`, `category`, `brand`, `model_type`, `purchase_date`, `purchase_price`, `location`) — tidak ada penerjemahan nama field antara draft dan Asset final. Field tanpa dasar di sumber **wajib** `"source": "empty"` dengan `"value": null` — dilarang diisi nilai karangan (guardrail `01-overview.md` §9).

```json
{
  "name":            { "value": "Dell Latitude 5450", "source": "extracted", "confidence": 0.97 },
  "category":        { "value": "IT Equipment > Laptop", "source": "inferred", "confidence": 0.72 },
  "brand":           { "value": "Dell", "source": "extracted", "confidence": 0.95 },
  "model_type":      { "value": "Latitude 5450", "source": "extracted", "confidence": 0.95 },
  "purchase_date":   { "value": "2026-08-20", "source": "extracted", "confidence": 0.93 },
  "purchase_price":  { "value": 18500000, "source": "extracted", "confidence": 0.88 },
  "location":        { "value": null, "source": "empty", "confidence": null }
}
```

`source` enum: `extracted` (ditemukan eksplisit di sumber) · `inferred` (disimpulkan AI dari konteks) · `empty` (tidak ditemukan, wajib dilengkapi user) · `edited` (diubah manual oleh user setelah ekstraksi — di-set FE saat `PATCH`).

### 4.3 Pemetaan Field → Asset

| Field ekstraksi | Target Asset | Catatan |
|-------------------|--------------|---------|
| Nama Aset | `asset.name` | Free text, tidak perlu resolve ke katalog manapun |
| Kategori | `asset.category` | Free text/pilihan sederhana |
| Merek | `asset.brand` | Free text |
| Model | `asset.model_type` | Free text |
| Tanggal Pembelian | `asset.purchase_date` | — |
| Harga Pembelian | `asset.purchase_price` | — |
| Lokasi | `asset.location` | Free text/pilihan sederhana |
| Asset ID | `asset.asset_id` | **Tidak diisi dari ekstraksi** — selalu auto-generate backend, format `AST-XXXXXX` |

### 4.4 API Response Contract

#### `POST /ai-registration/sessions` — Mulai sesi dari AI Chat Input (F-AIREG-01)

Request — multipart form, `rawText` dan `file` dua-duanya opsional tapi **minimal salah satu wajib ada** (composer gabungan):

```json
// teks saja
{ "rawText": "5 unit Dell Latitude 5450, dibeli 20 Agustus 2026 untuk tim desain" }
```

```json
// teks + file sekaligus dalam satu pesan — dikirim sebagai satu payload,
// AI menggabungkan keduanya jadi satu konteks ekstraksi (bukan dua proses terpisah)
{ "rawText": "ini invoice pembelian bulan lalu", "file": "<binary: invoice-agustus.pdf>" }
```

Response:

```json
{ "sessionId": "b3f1...", "status": "processing" }
```

#### `GET /ai-registration/sessions/{id}` — Poll status & ambil draft

```json
{
  "sessionId": "b3f1...",
  "status": "draft_ready",
  "hasText": true,
  "hasFile": false,
  "drafts": [
    {
      "id": "d1a2...",
      "sequenceNo": 1,
      "lowConfidence": false,
      "status": "pending",
      "fields": { "...": "lihat §4.2" }
    }
  ]
}
```

Response saat `status: "rejected"`:

```json
{ "sessionId": "b3f1...", "status": "rejected", "rejectReason": "irrelevant_document" }
```

#### `PATCH /ai-registration/drafts/{id}` — Edit field

```json
{ "fields": { "location": { "value": "Warehouse A", "source": "edited" } } }
```

#### `DELETE /ai-registration/drafts/{id}` — Hapus baris draft (row action `[🗑]`, F-AIREG-04)

Set `status: discarded` pada draft itu (bukan hard delete — tetap tercatat untuk audit). Response `204 No Content`. **Wajib dipanggil saat user klik `[🗑]`** — bukan cuma dihapus di state frontend, karena `confirm-all` memproses semua draft `status: pending` di sesi itu; draft yang tidak di-discard lewat endpoint ini akan tetap ikut diproses meski sudah "dihapus" secara visual di FE.

#### `POST /ai-registration/sessions/{id}/confirm-all` — Confirm & Create All (F-AIREG-05)

Diproses untuk semua draft `status: pending` dalam satu sesi (draft `discarded` dilewati) — baris `Ready` dibuat, baris `Missing fields` di-skip tanpa menggagalkan baris lain (partial success, `02-ui-design.md` §2.4).

```json
{
  "results": [
    { "draftId": "d1a2...", "status": "created", "assetId": "AST-000042", "createdAssetId": "a-8891..." },
    { "draftId": "d1a3...", "status": "created", "assetId": "AST-000043", "createdAssetId": "a-8892..." },
    { "draftId": "d1a4...", "status": "skipped", "reason": "missing_required_field", "missingFields": ["location"] }
  ],
  "summary": { "created": 2, "skipped": 1 }
}
```

> Field nullable mengikuti null rendering C-2 §8. Error response mengikuti `error-handling-convention.md` (C-2) standar — tidak didefinisikan ulang di sini.

#### `GET /assets` — Asset List (F-AIREG-09)

Query params:

| Param | Type | Deskripsi |
|-------|------|-----------|
| `search` | String | Cari di Name, Asset ID, Category, Brand, Model/Type, Location |
| `category` | String[] | Filter multi-select |
| `brand` | String[] | Filter multi-select |
| `location` | String[] | Filter multi-select |
| `purchaseDateFrom` / `purchaseDateTo` | Date | Filter range Purchase Date |
| `sort` | String | Nama kolom, default `createdAt` |
| `order` | Enum | `asc` \| `desc`, default `desc` |
| `page` | Integer | Default `1` |
| `pageSize` | Integer | Default `10`; pilihan `10`/`25`/`50`/`100`/`all` |

```json
{
  "assets": [
    {
      "id": "a-8891...",
      "assetId": "AST-000042",
      "name": "Dell Latitude 5450",
      "category": "IT Equipment > Laptop",
      "brand": "Dell",
      "modelType": "Latitude 5450",
      "purchaseDate": "2026-08-20",
      "purchasePrice": 18500000,
      "location": "Warehouse A",
      "createdAt": "2026-08-28T10:00:00Z"
    }
  ],
  "pagination": { "page": 1, "pageSize": 10, "totalItems": 1, "totalPages": 1 }
}
```

#### `GET /assets/download` — Download Asset List (F-AIREG-09)

Query params sama dengan `GET /assets` (tanpa `page`/`pageSize` — download seluruh hasil yang match filter/search) + `format`: `csv` \| `xlsx`. Response: file `Assets.csv` / `Assets.xlsx`.

#### `POST /assets` — Manual Form Submit (F-AIREG-08)

Request:

```json
{
  "name": "Dell Latitude 5450",
  "category": "IT Equipment > Laptop",
  "brand": "Dell",
  "modelType": "Latitude 5450",
  "purchaseDate": "2026-08-20",
  "purchasePrice": 18500000,
  "location": "Warehouse A"
}
```

Response — sama shape dengan hasil `confirm-all` per baris:

```json
{ "assetId": "AST-000044", "createdAssetId": "a-8893..." }
```

#### `PATCH /assets/{id}` — Edit Asset (F-AIREG-10)

Request — field yang mau diubah saja (`asset_id` tidak boleh dikirim/diubah):

```json
{ "location": "Warehouse B", "purchasePrice": 17500000 }
```

Response — shape sama dengan `GET /assets` per item.

#### `DELETE /assets/{id}` — Delete Asset (F-AIREG-11)

Tidak ada request body. Response `204 No Content` saat berhasil. Error mengikuti `error-handling-convention.md` (C-2) standar.

---

## Changelog

| Tanggal | Perubahan | Oleh |
|---------|-----------|------|
| 2026-08-28 | Initial version — entitas session/draft, struktur fields JSON dengan source tracking, pemetaan ke Asset, API contract | PRD Builder |
| 2026-08-28 | Simplifikasi: hapus Serial Number, hapus `asset_master_id`, entitas Asset jadi sederhana/berdiri sendiri (bukan produksi), Confirm & Create selalu langsung buat aset (tanpa approval) | PRD Builder |
| 2026-08-28 | Endpoint confirm diganti jadi bulk per-sesi (`confirm-all`) dengan hasil per baris (created/skipped) — selaras UI tabel multi-baris di halaman yang sama | PRD Builder |
| 2026-08-28 | Tambah query params `GET /assets` (search/filter/sort/pagination), endpoint `GET /assets/download`, dan `POST /assets` untuk Manual Form Submit | PRD Builder |
| 2026-08-28 | Tambah `PATCH /assets/{id}` (Edit) dan `DELETE /assets/{id}` (Delete, single row) | PRD Builder |
| 2026-08-28 | "Save as Draft" dihapus dari MVP — F-AIREG di-renumber, referensi endpoint disesuaikan | PRD Builder |
| 2026-08-28 | Fix key JSON draft `asset_name`→`name` (samakan dengan entitas Asset); tambah `DELETE /ai-registration/drafts/{id}` untuk discard row (menutup celah logic: row yang dihapus di FE tapi tidak di-discard di backend akan tetap ke-create saat confirm-all) | PRD Builder |
