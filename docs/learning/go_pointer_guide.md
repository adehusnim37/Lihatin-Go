# Panduan Pointers di Golang & GORM

Dokumen ini menjelaskan kapan harus menggunakan Pointer (`*Type`) dan kapan menggunakan Nilai Biasa (`Type`), khususnya dalam konteks DTO (Data Transfer Object) dan Interaksi Database dengan GORM.

## 1. Konsep Dasar: Zero Value vs Nil

Di Golang, setiap tipe data memiliki "Zero Value" (Nilai Kosong) bawaan jika tidak diinisialisasi.

| Tipe Data   | Kode                       | Zero Value           |
| :---------- | :------------------------- | :------------------- |
| String      | `string`                   | `""` (string kosong) |
| Integer     | `int`                      | `0`                  |
| Boolean     | `bool`                     | `false`              |
| **Pointer** | `*string`, `*int`, `*bool` | **`nil`**            |

### Masalah "Zero Value"

Kita sering kesulitan membedakan antara:

1.  User **sengaja** mengirim nilai kosong (misal: `false` atau `0`).
2.  User **tidak mengirim** data apa-apa (sehingga Go mengisinya dengan default `false` atau `0`).

## 2. Studi Kasus: Boolean Flag (`EnableStats`)

Bayangkan kita punya field JSON `enable_stats`.

### Skenario A: Menggunakan Nilai Biasa (`bool`)

```go
type Request struct {
    EnableStats bool `json:"enable_stats"`
}
```

- **JSON:** `{"enable_stats": false}` -> **Go:** `false` (User sengaja mematikan).
- **JSON:** `{}` (Kosong) -> **Go:** `false` (Default Go).

**Masalah:** Kita tidak tahu apakah user ingin mematikan statistik atau user lupa mengirim data (padahal default aplikasi mungkin ingin statistik NYALA/TRUE).

### Skenario B: Menggunakan Pointer (`*bool`)

```go
type Request struct {
    EnableStats *bool `json:"enable_stats"`
}
```

- **JSON:** `{"enable_stats": false}` -> **Go:** `pointer ke false` (Ada isinya).
- **JSON:** `{}` (Kosong) -> **Go:** `nil` (Kosong beneran).

**Solusi:** Dengan pointer, kita bisa tahu kalau datanya `nil`.

- Jika `nil` -> Kita bisa set Default Value aplikasi (`true`).
- Jika `false` -> Kita hormati pilihan user (`false`).

## 3. Prilaku GORM (Database)

GORM memiliki sifat unik: **Secara default, GORM akan MENGABAIKAN field yang bernilai Zero Value saat Create/Update.**

Misal struct Model:

```go
type User struct {
    IsActive bool // zero value: false
}
```

Jika kamu simpan objek `{IsActive: false}`:

- GORM melihat: _"Wah field ini nilainya `false` (zero value), skip aja deh, gak usah dimasukin query SQL."_
- **Akibat:** Database akan menggunakan **DEFAULT VALUE** kolom tersebut (misal di DB default-nya `1` / `TRUE`).
- **Bug:** User kirim `false` (tidak aktif), tapi di DB tetap `TRUE` (aktif).

**Solusi GORM:** Gunakan Pointer `*bool`.

- Jika pointer berisi `&false`, GORM tahu itu bukan `nil` (bukan kosong), jadi GORM akan memaksa simpan `FALSE` ke database.

## 4. Kapan Pakai Pointer vs Biasa? (Rule of Thumb)

| Jenis Field                 | Gunakan                         | Alasan                                                                               |
| :-------------------------- | :------------------------------ | :----------------------------------------------------------------------------------- |
| **Wajib Diisi (Required)**  | Nilai Biasa (`string`, `int`)   | Validator menjamin data pasti ada. Tidak mungkin `nil`. Lebih simpel.                |
| **Opsional (Optional)**     | **Pointer** (`*string`, `*int`) | Supaya bisa membedakan antara "User kirim nilai 0/false" vs "User tidak kirim data". |
| **Flag Boolean (IsActive)** | **Pointer** (`*bool`)           | Agar GORM bisa menyimpan nilai `false` ke database dan tidak kena skip.              |

## 5. Penjelasan Helper Kita

### `helpers.PtrToValue`

Kita membuat helper ini untuk menangani Pointers yang punya Default Value.

```go
// Logic manual yang ribet
enableStats := true // Default maunya TRUE
if link.EnableStats != nil {
    enableStats = *link.EnableStats // Ambil nilai user kalau ada
}

// Dengan Helper
enableStats := helpers.PtrToValue(link.EnableStats, true)
```

Ini aman dari **Panic** (Nil Pointer Dereference) dan kodenya jauh lebih rapi.

### Kasus Khusus: `UserID string` ke `*string`

Di kasus Repository `UserID`:

- **DTO:** `UserID string` (Bukan pointer, karena kita terima string kosong).
- **Model DB:** `UserID *string` (Pointer, karena kolom DB boleh `NULL`).

Logic-nya terbalik: Kita punya value, mau dijadiin pointer (atau null).

```go
var userIDPtr *string
if link.UserID != "" {
    userIDPtr = &link.UserID // String ada isi -> Bikin pointer
}
// else biarkan nil (String kosong -> Jadi NULL di DB)
```

Ini tidak pakai helper `PtrToValue` karena arah konversinya sebaliknya (Value -> Pointer).

## 6. JSON Tag `omitempty` vs Database

Sering ada kesalahpahaman bahwa tag `json:"...,omitempty"` mempengaruhi cara data disimpan di database.
**Kenyataannya:** Tag `json` HANYA mempengaruhi format JSON saat:
1.  **Serialization (Marshal):** Mengubah struct Go -> JSON string (Response API).
2.  **Deserialization (Unmarshal):** Mengubah JSON string -> struct Go (Request Body).

Tag ini **TIDAK** ada hubungannya dengan GORM atau tabel database. Database hanya peduli dengan tag `gorm:"..."`.

### Keamanan Non-Pointer Field (String/Int Biasa)

Apakah aman menggunakan `omitempty` tanpa pointer?

```go
type Request struct {
    Title string `json:"title,omitempty"`
}
```

**Jawabannya: AMAN.**

*   **Logic:** Tipe data non-pointer (`string`, `int`, `bool`) **TIDAK BISA NIL**.
*   Jika JSON tidak mengirim field `title`, Go akan mengisinya dengan **Zero Value**:
    *   `string` -> `""` (String kosong)
    *   `int` -> `0`
*   **Kenapa Aman?** Karena tidak bisa `nil`, maka **tidak mungkin terjadi Panic `nil pointer dereference`** saat akses `req.Title`. Kita tidak perlu cek `if req.Title != nil`.

### Prilaku GORM pada Non-Pointer Field

Bagaimana jika kita simpan `Title` kosong tadi ke Database?

```go
// Model
type ShortLink struct {
    Title string `gorm:"size:255"` // Bukan pointer
}

// Repository
db.Create(&ShortLink{Title: req.Title}) // req.Title = ""
```

*   **GORM `Create`:** Aman. GORM akan menyimpan string kosong `""` ke kolom `title` di tabel.
*   **GORM `Updates`:** Hati-hati. Kalau pakai struct biasa, GORM akan menganggap string kosong `""` sebagai "tidak ada perubahan" (zero value di-skip).

**Kesimpulan:**
Untuk field seperti `Title` atau `Description` yang boleh berisi string kosong, menggunakan tipe data biasa (`string`) itu **AMAN, LEBIH SIMPEL, dan TIDAK PERLU PENGECEKAN NIL**.

