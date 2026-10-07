# 🏢 QTERA Integrated Chatbot v2 & IT Asset Management Platform
> **Dokumentasi Sistem & Panduan Operasional (Non-Technical IT / Management Overview)**  
> Platform Manajemen Aset IT Enterprise terintegrasi dengan Agentic AI Copilot (Natural Language, Voice Dictation, dan OCR/Document Ingestion).

---

## 📌 1. Ringkasan Eksekutif & Nilai Bisnis

**QTERA Integrated Chatbot v2** adalah solusi pengelolaan siklus hidup aset IT (*IT Asset Lifecycle Management*) yang dipadukan dengan asisten kecerdasan buatan (*Agentic AI Copilot*). 

Sistem ini mentransformasi proses inventarisasi konvensional yang manual menjadi interaksi percakapan (*conversational interface*), memungkinkan pengguna mengelola data aset cukup melalui perintah teks (chat) maupun suara (voice).

### 🎯 Nilai Tambah & Kemampuan Utama:
- **Conversational Asset Operations**: Pengguna dapat mencari, menambah, memperbarui, dan memeriksa status ketersediaan barang melalui chat interaktif atau dikte suara.
- **Multimodal Document Processing**: Otomatisasi ekstraksi data dari faktur, dokumen pembelian, atau spesifikasi teknis (PDF & Gambar via OCR Tesseract) langsung ke sistem.
- **Multi-Tenant Data Segregation**: Pemisahan data aset secara ketat antar-perusahaan/entitas bisnis (*company-level isolation*).
- **Role-Based Access Control (RBAC)**: Pembagian hak akses berjenjang (*Admin*, *Operator*, *Viewer*) untuk menjaga integritas data dan kepatuhan audit.
- **Omnichannel Access**: Dapat diakses melalui Web Dashboard (Desktop/Laptop) maupun Aplikasi Mobile (Android/iOS).

---

## 🏗️ 2. Arsitektur Sistem Tingkat Tinggi

Sistem dibangun menggunakan pendekatan **Decoupled Multi-Service Architecture**, di mana setiap modul memiliki tanggung jawab spesifik dan berkomunikasi melalui protokol standar (HTTP REST, WebSocket, dan Model Context Protocol):

```mermaid
flowchart TB
    subgraph Client_Tier ["Client Tier (Antarmuka Pengguna)"]
        WebUI["Web Portal (React JS + Vite)<br/>Port: 5173"]
        MobileApp["Mobile App (React Native / Expo)"]
    end

    subgraph Service_Tier ["Application & AI Tier"]
        CoreAPI["Core Backend Server (Go Fiber)<br/>Port: 3000<br/>• Auth & RBAC<br/>• WebSocket Hub<br/>• Speech-to-Text"]
        AIOrchestrator["AI Orchestrator (eiai_go)<br/>Port: 8000<br/>• Multi-Turn Reasoning Agent<br/>• MCP Tool Calling<br/>• Document & OCR Extractor"]
    end

    subgraph Data_AI_Tier ["Data & Inference Layer"]
        Postgres[("Database PostgreSQL<br/>Port: 5432<br/>(poc_ai)")]
        LLMProvider["External LLM Inference API<br/>(DeepSeek / OpenRouter)"]
    end

    WebUI -->|"REST & WebSocket"| CoreAPI
    MobileApp -->|"REST & WebSocket"| CoreAPI
    CoreAPI -->|"Query & Transaksi"| Postgres
    CoreAPI <-->|"RAG & Conversational Forwarding"| AIOrchestrator
    AIOrchestrator -->|"Inference & Prompt Reasoning"| LLMProvider
    AIOrchestrator -->|"Action Tools via MCP"| CoreAPI
```

---

## 🧩 3. Komponen Utama Layanan

| Komponen | Direktori | Port / Engine | Peran & Deskripsi |
| :--- | :--- | :--- | :--- |
| **Web Frontend** | `pocAI2/ai-registration-frontend` | `5173` (Vite) | Antarmuka web utama untuk dashboard aset, tabel inventaris, dan antarmuka obrolan AI. |
| **Mobile App** | `pocAI2/react_native/eiai_react` | Expo Metro | Antarmuka mobile berbasis React Native untuk akses lapangan dan input suara langsung. |
| **Core Backend** | `pocAI2/server/GO` | `3000` (Go Fiber) | Layanan inti untuk autentikasi pengguna (JWT), WebSocket, transaksi inventaris, dan koneksi database. |
| **AI Orchestrator** | `pocAI2/eiai_go` | `8000` (Go) | Mesin AI yang bertugas menerjemahkan maksud pengguna (*intent*), mengekstrak dokumen, dan memanggil fungsi/tools sistem (*MCP*). |
| **Database** | PostgreSQL | `5432` | Basis data relasional (*schema: `poc_ai`*) untuk master aset, log percakapan, transaksi peminjaman, dan jadwal audit. |

---

## 📋 4. Prasyarat Lingkungan Sistem (Prerequisites - 100% Fitur)

Agar **seluruh fitur sistem dapat berjalan 100%** (termasuk Text Chatbot, AI Agent Tools, Voice Input/STT, TTS, dan OCR Dokumen/Gambar), pastikan perangkat telah terpasang:

| Komponen / Tool | Versi / Tipe | Peran & Penggunaan |
| :--- | :--- | :--- |
| **Go (Golang)** | `1.21+` | Menjalankan Core Backend (`server/GO`), AI Orchestrator (`eiai_go`), dan `mcp_server`. |
| **Node.js & npm** | `18.x+ LTS` | Menjalankan Web Portal (`ai-registration-frontend`) dan Mobile App. |
| **PostgreSQL** | `14+` | Database relasional utama (schema: `poc_ai`, port: `5432`). |
| **Tesseract OCR** | `5.x+` | **Wajib untuk OCR Gambar/Faktur**: Membaca teks foto kwitansi/faktur fisik (`.png`, `.jpg`, `.jpeg`, `.webp`). |
| **FFmpeg** | Versi terbaru | **Wajib untuk Input Suara**: Pra-pemrosesan & filter background noise audio Speech-to-Text. |
| **API Key LLM** | DeepSeek / OpenRouter | Otak penalaran AI Agent & pemanggilan MCP Tools. |
| **Voice API Key** *(Opsional)* | ElevenLabs / Cartesia | Untuk fitur transkripsi suara (STT) dan respon suara (TTS). |

#### ⚡ Instalasi Cepat Prasyarat di Windows (via PowerShell Administrator):
```powershell
winget install GoLang.Go
winget install OpenJS.NodeJS.LTS
winget install PostgreSQL.PostgreSQL
winget install UB-Mannheim.TesseractOCR
winget install Gyan.FFmpeg
```
> [!IMPORTANT]
> - **Tesseract OCR**: Saat proses instalasi installer Windows, pastikan mencentang paket bahasa tambahan **Indonesian (`ind`)** dan **English (`eng`)**, serta pastikan path binary `tesseract.exe` (misal: `C:\Program Files\Tesseract-OCR`) terdaftar di Environment Variable `PATH`.
> - **FFmpeg**: Pastikan perintah `ffmpeg` dapat dipanggil langsung dari terminal PowerShell/CMD baru.

---

## 🗄️ 5. Inisialisasi Basis Data (Database Setup)

Sebelum menjalankan aplikasi, inisialisasi basis data PostgreSQL lokal:

1. Buka terminal atau konsol `psql`, lalu buat database baru:
   ```sql
   CREATE DATABASE poc_ai;
   ```
2. Jalankan skrip konsolidasi skema tabel dan data awal (*seed data*) dari folder `migrations/`:
   ```bash
   # Di root direktori project:
   psql -U postgres -d poc_ai -f migrations/000000_init_all_schema.sql
   psql -U postgres -d poc_ai -f migrations/000011_seed_initial_data.up.sql
   ```
   *(Opsional: Jika ada skenario anomali peminjaman, dapat mengeksekusi `migrations/000012_create_borrow_anomaly_tables.sql` dan `migrations/000013_seed_borrow_anomaly_scenarios.sql`).*

---

## ⚙️ 6. Konfigurasi Variabel Lingkungan (`.env`)

Karena file `.env` diabaikan oleh Git demi alasan keamanan, buat file konfigurasi berikut secara manual:

### 1. Konfigurasi AI Orchestrator (`pocAI2/eiai_go/.env`)
File ini mengatur koneksi AI Orchestrator ke Core Backend dan Provider LLM:
```env
BASE_URL=http://127.0.0.1:3000/api
PORT=8000

# Provider LLM (DeepSeek / OpenRouter)
BASE_AI_URL=https://api.deepseek.com
AI_KEY=sk-your-ai-api-key-here
```

### 2. Konfigurasi Core Backend (`pocAI2/server/GO/.env`)
File ini mengatur koneksi database dan integrasi audio/voice:
```env
# Koneksi Database PostgreSQL
DATABASE_URL=postgresql://postgres:password@localhost:5432/poc_ai
DB_HOST=localhost
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=poc_ai
DB_PORT=5432
DB_SSLMODE=disable

# Keamanan JWT
JWT_SECRET=qtera-super-secret-jwt-key-poc-2026

# Integrasi Suara & Audio (Opsional / Untuk Fitur Voice 100%)
ELEVENLABS_API_KEY=your_elevenlabs_api_key
ELEVENLABS_SCRIBE_MODEL=scribe_v2
CARTESIA_KEY=your_cartesia_key
CARTESIA_MODEL=sonic-3.6
CARTESIA_VOICE_ID=b441c4fd-4910-4c55-ae56-f0291057e2cc
```

---

## 🚀 7. Panduan Menjalankan Layanan (Runbook)

Karena sistem ini berbasis *decoupled multi-service*, **setiap service dijalankan di jendela terminal yang terpisah** agar proses dapat berjalan bersamaan (*concurrently*):

### Tahap 1: Unduh Seluruh Dependensi

Buka terminal dan unduh paket di tiap direktori modul:

1. **Dependensi Frontend Web**:
   ```bash
   cd pocAI2/ai-registration-frontend
   npm install
   ```
2. **Dependensi Mobile App** *(Opsional)*:
   ```bash
   cd pocAI2/react_native/eiai_react
   npm install
   npx expo install expo-speech
   ```
3. **Dependensi Backend & AI Service (Go)**:
   ```bash
   cd pocAI2/server/GO
   go mod download

   cd ../../eiai_go
   go mod download

   cd ../mcp_server
   go mod download
   ```

---

### Tahap 2: Menjalankan Seluruh Service

Buka **3 jendela terminal** secara terpisah:

#### 🖥️ Terminal 1: Core Backend Server
```bash
cd pocAI2/server/GO
go run main.go
```
- **Keluaran Sukses**: Notifikasi Go Fiber aktif di `http://127.0.0.1:3000`.

#### 🧠 Terminal 2: AI Orchestrator Service
```bash
cd pocAI2/eiai_go
go run main.go
```
- **Keluaran Sukses**: Layanan AI Orchestrator mendengarkan di port `8000` dan secara otomatis menginisialisasi sub-proses `mcp_server`.

#### 🌐 Terminal 3: Web Client (Frontend)
```bash
cd pocAI2/ai-registration-frontend
npm run dev
```
- **Akses Pengguna**: Buka browser di [http://localhost:5173](http://localhost:5173).

#### 📱 Terminal 4 (Opsional): Mobile Application
```bash
cd pocAI2/react_native/eiai_react
npm start
```
- **Akses Pengguna**: Pindai QR Code yang tampil di konsol menggunakan aplikasi Expo Go pada perangkat Android/iOS.

---

## 👥 8. Matriks Pengguna & Hak Akses (Default Seed Users)

Untuk keperluan pengujian alur fungsional dan RBAC, berikut daftar akun uji coba yang telah tersedia pada basis data:

| Username | Role | Entitas Perusahaan | Cakupan Wewenang |
| :--- | :--- | :--- | :--- |
| **admin** | `admin` | Qtera Mandiri | **Akses Penuh**: Manajemen seluruh aset, manajemen pengguna, approval peminjaman, dan konfigurasi master data. |
| **tordalk** | `operator` | Qtera Mandiri | **Operasional**: Input aset baru, pembaharuan status fisik, pencatatan peminjaman dan pengembalian barang. |
| **jason** | `viewer` | Bunda Mulia | **Read-Only**: Monitoring ketersediaan dan pencarian data aset terbatas pada lingkup perusahaannya. |
| **andhika** | `operator` | Adara Group | **Operasional**: Pengelolaan aset khusus untuk entitas Adara Group. |

*(Anda juga dapat membuat pengguna baru langsung melalui tombol **Register** pada halaman login Web).*

---

## 💬 9. Skenario Uji 100% Fitur Interaksi AI

Setelah login ke aplikasi web di [http://localhost:5173](http://localhost:5173), uji seluruh skenario fitur berikut:

1. **Pencarian & Audit Stok (Natural Language Chat)**:
   - *"Tampilkan seluruh inventaris laptop kategori Dell yang statusnya aktif."*
   - *"Berapa unit monitor yang saat ini tersedia di Ruang Server?"*
2. **Pengecekan Status Peminjaman & Mutasi Aset**:
   - *"Apakah aset dengan ID AST-5DFBE7CF sedang dipinjam?"*
   - *"Tampilkan daftar transaksi peminjaman yang berstatus pending approval."*
3. **Ekstraksi Dokumen & Faktur Gambar (Multimodal OCR 100%)**:
   - Lampirkan file invoice digital (PDF) atau foto nota/faktur fisik (`.png`, `.jpg`) via tombol attachment paperclip di chat.
   - Ketik: *"Ekstrak data barang dari dokumen ini dan daftarkan sebagai aset baru."*
   - Tesseract OCR dan parser dokumen akan mengekstrak detail barang, harga, serta tanggal pembelian secara otomatis.
4. **Input Suara & Audio Dictation (Voice Input & STT 100%)**:
   - Klik ikon mikrofon pada kolom obrolan web atau mobile app, lalu rekam instruksi suara Anda.
   - FFmpeg akan melakukan denoising audio dan mengirimkannya ke engine STT untuk dieksekusi oleh AI Agent.

---

## 🔧 10. Pemeriksaan Kesehatan Sistem & Troubleshooting

| Gejala Masalah | Indikasi Penyebab | Langkah Solusi |
| :--- | :--- | :--- |
| **Chatbot tidak merespons atau mengembalikan timeout** | 1. Service `eiai_go` belum aktif.<br/>2. Kunci `AI_KEY` pada `.env` kedaluwarsa / saldo kuota habis. | Pastikan Terminal 2 berjalan lancar dan periksa validitas `AI_KEY` pada `pocAI2/eiai_go/.env`. |
| **Daftar aset tidak muncul di antarmuka Web** | 1. Service `server/GO` belum berjalan.<br/>2. Koneksi ke database PostgreSQL gagal. | Periksa log Terminal 1 (`server/GO`) dan pastikan service PostgreSQL aktif di port `5432`. |
| **Error: `Tesseract OCR binary not found in PATH` saat upload gambar** | Tesseract OCR belum terinstal atau path belum ditambahkan ke System Environment Variables. | Install via `winget install UB-Mannheim.TesseractOCR`, lalu restart terminal agar PATH terbaca. |
| **Audio filter warning: `ffmpeg filter notice`** | Binary `ffmpeg` belum terpasang di PATH sistem. | Install via `winget install Gyan.FFmpeg`. (Sistem tetap memproses audio asli, namun filtering noise dinonaktifkan). |
| **Port Conflict (`listen tcp :3000 or :8000: bind: address already in use`)** | Port tersebut sedang digunakan oleh proses lain di latar belakang. | Hentikan proses yang berjalan di port terkait atau gunakan perintah `taskkill /F /IM <nama_proses>.exe` sebelum menjalankan ulang. |
| **Menghentikan Layanan** | Menutup proses aplikasi secara tertib (*graceful shutdown*). | Tekan kombinasi keyboard `Ctrl + C` pada masing-masing jendela terminal yang aktif. |

---

*Untuk integrasi tingkat lanjut, implementasi MCP tools baru, atau skema arsitektur mendalam, silakan merujuk ke dokumen teknis lengkap di [TECHNICAL_DOCUMENTATION.md](file:///c:/Users/user/OneDrive/Documents/proyek_DH/QTERA/intergrated_chatbot_v2/intergrated_chatbot_v2/pocAI2/TECHNICAL_DOCUMENTATION.md).*
