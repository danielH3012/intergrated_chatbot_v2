# Database Migrations Documentation

Folder ini berisi script migrasi SQL untuk inisialisasi dan pengelolaan skema basis data PostgreSQL (`poc_ai`) untuk proyek **QTERA Mandiri / POC AI**.

---

## 📂 Struktur File Migrasi

| Nomor | Up File | Down File | Keterangan |
| :--- | :--- | :--- | :--- |
| **000000** | `000000_init_all_schema.sql` | — | Skrip konsolidasi seluruh skema tabel & relasi |
| **000001** | `000001_create_schema_and_extensions.up.sql` | `.down.sql` | Ekstensi `pgcrypto` & skema `schema1` |
| **000002** | `000002_create_perusahaan_table.up.sql` | `.down.sql` | Tabel `public.perusahaan` (multi-tenant) |
| **000003** | `000003_create_legacy_user_table.up.sql` | `.down.sql` | Tabel `public."user"` (referensi relasi foreign key) |
| **000004** | `000004_create_users_table.up.sql` | `.down.sql` | Tabel `public.users` (autentikasi & session login) |
| **000005** | `000005_create_assets_table.up.sql` | `.down.sql` | Tabel `public.assets` (katalog perangkat & inventaris) |
| **000006** | `000006_create_chat_table.up.sql` | `.down.sql` | Tabel `public.chat` (riwayat percakapan & lampiran) |
| **000007** | `000007_create_schedule_table.up.sql` | `.down.sql` | Tabel `public.schedule` (jadwal audit & inspeksi rutin) |
| **000008** | `000008_create_tools_history_table.up.sql` | `.down.sql` | Tabel `public.tools_history` (audit trail eksekusi AI tools) |
| **000009** | `000009_create_groups_table.up.sql` | `.down.sql` | Tabel `public.groups` (manajemen grup & tim) |
| **000010** | `000010_create_ai_registration_tables.up.sql` | `.down.sql` | Tabel `ai_registration_sessions` & `ai_registration_drafts` |
| **000011** | `000011_seed_initial_data.up.sql` | `.down.sql` | Data awal perusahaan, user default, dan aset sampel |

---

## 🚀 Cara Menjalankan Migrasi

### Opsi 1: Menjalankan Seluruh Skema Sekaligus (Direkomendasikan)
Gunakan file `000000_init_all_schema.sql` atau `database.sql`:

```bash
# Menggunakan psql
psql -U postgres -d poc_ai -f migrations/000000_init_all_schema.sql
psql -U postgres -d poc_ai -f migrations/000011_seed_initial_data.up.sql
```

Atau menggunakan root `database.sql` (berisi skema + seed data):
```bash
psql -U postgres -d poc_ai -f database.sql
```

---

### Opsi 2: Menjalankan Migrasi Langkah-per-Langkah (Step-by-Step)
Jalankan file `.up.sql` secara berurutan:

```bash
psql -U postgres -d poc_ai -f migrations/000001_create_schema_and_extensions.up.sql
psql -U postgres -d poc_ai -f migrations/000002_create_perusahaan_table.up.sql
psql -U postgres -d poc_ai -f migrations/000003_create_legacy_user_table.up.sql
psql -U postgres -d poc_ai -f migrations/000004_create_users_table.up.sql
psql -U postgres -d poc_ai -f migrations/000005_create_assets_table.up.sql
psql -U postgres -d poc_ai -f migrations/000006_create_chat_table.up.sql
psql -U postgres -d poc_ai -f migrations/000007_create_schedule_table.up.sql
psql -U postgres -d poc_ai -f migrations/000008_create_tools_history_table.up.sql
psql -U postgres -d poc_ai -f migrations/000009_create_groups_table.up.sql
psql -U postgres -d poc_ai -f migrations/000010_create_ai_registration_tables.up.sql
psql -U postgres -d poc_ai -f migrations/000011_seed_initial_data.up.sql
```

---

### Opsi 3: Rollback / Pembatalan Migrasi
Untuk menghapus tabel atau data tertentu, jalankan file `.down.sql` terkait dari urutan belakang (reverse order):

```bash
psql -U postgres -d poc_ai -f migrations/000011_seed_initial_data.down.sql
psql -U postgres -d poc_ai -f migrations/000010_create_ai_registration_tables.down.sql
# ... dan seterusnya
```

---

### Opsi 4: Restore dari Custom Dump Binary (`db.sql`)
Jika ingin memulihkan langsung dari backup binary `db.sql` asli:

```bash
pg_restore -U postgres -d poc_ai -v db.sql
```
