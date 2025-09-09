# Sistem Validasi Comprehensive untuk Gin + validator.v10

## Overview

Sistem validasi ini menyediakan utilitas lengkap untuk menangani berbagai jenis error validasi dalam aplikasi Gin dengan validator.v10, termasuk:

- ✅ **JSON Syntax Errors** - Error parsing JSON
- ✅ **Type Mismatch Errors** - Error tipe data
- ✅ **Validation Errors** - Error aturan validasi
- ✅ **Custom Validators** - Validator khusus
- ✅ **Indonesian Error Messages** - Pesan error dalam Bahasa Indonesia
- ✅ **Structured Response** - Response terstruktur dengan details

## Fitur Utama

### 1. **Custom Validators**

| Validator | Deskripsi | Contoh |
|-----------|-----------|---------|
| `pwdcomplex` | Password kompleks (min 8 char, uppercase, lowercase, number, symbol) | `binding:"pwdcomplex"` |
| `username` | Username format (alphanumeric, underscore, hyphen only) | `binding:"username"` |
| `unique` | Array tidak boleh duplikat | `binding:"unique"` |
| `no_space` | Tidak boleh mengandung spasi | `binding:"no_space"` |

### 2. **Error Types Handling**

#### JSON Syntax Error
```json
{
  "success": false,
  "message": "Validasi gagal",
  "errors": {
    "json": "Format JSON tidak valid"
  },
  "details": [
    {
      "field": "json",
      "message": "Format JSON tidak valid pada posisi 15"
    }
  ]
}
```

#### Type Mismatch Error
```json
{
  "success": false,
  "message": "Validasi gagal", 
  "errors": {
    "codes": "codes harus bertipe array teks"
  },
  "details": [
    {
      "field": "codes",
      "message": "codes harus bertipe array teks"
    }
  ]
}
```

#### Validation Error
```json
{
  "success": false,
  "message": "Validasi gagal",
  "errors": {
    "Password": "Kata Sandi harus mengandung minimal 8 karakter, huruf besar, huruf kecil, angka, dan simbol"
  },
  "details": [
    {
      "field": "Password", 
      "message": "Kata Sandi harus mengandung minimal 8 karakter, huruf besar, huruf kecil, angka, dan simbol"
    }
  ]
}
```

## Setup dan Penggunaan

### 1. **Setup Custom Validators**

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/gin-gonic/gin/binding"
    "github.com/go-playground/validator/v10"
    "github.com/your-project/utils"
)

func main() {
    // Setup custom validators
    if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
        utils.SetupCustomValidators(v)
    }
    
    r := gin.Default()
    // ... rest of your setup
}
```

### 2. **Struct Definition dengan Label**

```go
type UserRegistrationRequest struct {
    Username string `json:"username" label:"Nama Pengguna" binding:"required,min=3,max=20,username"`
    Email    string `json:"email" label:"Email" binding:"required,email"`
    Password string `json:"password" label:"Kata Sandi" binding:"required,pwdcomplex"`
}

type BulkDeleteRequest struct {
    Codes []string `json:"codes" label:"Kode Short Link" binding:"required,min=1,max=100,unique,dive,required,no_space,alphanum"`
}
```

### 3. **Handler Implementation**

```go
func HandleUserRegistration(c *gin.Context) {
    var req UserRegistrationRequest
    
    if err := c.ShouldBindJSON(&req); err != nil {
        utils.SendValidationError(c, err, &req)
        return
    }
    
    // Process valid request...
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "message": "Registrasi berhasil",
        "data": req,
    })
}
```

## Tag Validation Supported

### Built-in Validators
- `required` - Field wajib diisi
- `min=X` - Minimal X karakter/item
- `max=X` - Maksimal X karakter/item
- `len=X` - Tepat X karakter/item
- `email` - Format email valid
- `url` - Format URL valid
- `alphanum` - Hanya huruf dan angka
- `alpha` - Hanya huruf
- `numeric` - Hanya angka
- `oneof=a b c` - Salah satu dari nilai yang diberikan
- `gte=X` - Greater than or equal
- `lte=X` - Less than or equal
- `gt=X` - Greater than
- `lt=X` - Less than

### Custom Validators
- `pwdcomplex` - Password kompleks
- `username` - Username format
- `unique` - Array unique
- `no_space` - Tidak boleh spasi

### Array/Slice Validation
- `dive` - Validasi setiap elemen dalam array
- `unique` - Array tidak boleh duplikat

## Contoh Response untuk Berbagai Skenario

### 1. Body Kosong
**Request:** (empty body)
**Response:**
```json
{
  "success": false,
  "message": "Validasi gagal",
  "errors": {
    "request": "Format request tidak valid"
  },
  "details": [
    {
      "field": "request",
      "message": "Format request tidak valid"
    }
  ]
}
```

### 2. Array Kosong
**Request:** `{"codes": []}`
**Response:**
```json
{
  "success": false,
  "message": "Validasi gagal",
  "errors": {
    "Codes": "Kode Short Link minimal 1 karakter/item"
  },
  "details": [
    {
      "field": "Codes",
      "message": "Kode Short Link minimal 1 karakter/item"
    }
  ]
}
```

### 3. Array Duplikat
**Request:** `{"codes": ["abc123", "def456", "abc123"]}`
**Response:**
```json
{
  "success": false,
  "message": "Validasi gagal",
  "errors": {
    "Codes": "Kode Short Link tidak boleh ada yang duplikat"
  },
  "details": [
    {
      "field": "Codes",
      "message": "Kode Short Link tidak boleh ada yang duplikat"
    }
  ]
}
```

### 4. Tipe Data Salah
**Request:** `{"codes": "not-an-array"}`
**Response:**
```json
{
  "success": false,
  "message": "Validasi gagal",
  "errors": {
    "codes": "codes harus bertipe array teks"
  },
  "details": [
    {
      "field": "codes",
      "message": "codes harus bertipe array teks"
    }
  ]
}
```

### 5. Item dengan Spasi
**Request:** `{"codes": ["abc 123", "def456"]}`
**Response:**
```json
{
  "success": false,
  "message": "Validasi gagal",
  "errors": {
    "Codes[0]": "Codes[0] tidak boleh mengandung spasi"
  },
  "details": [
    {
      "field": "Codes[0]",
      "message": "Codes[0] tidak boleh mengandung spasi"
    }
  ]
}
```

### 6. Password Tidak Kompleks
**Request:** `{"password": "simple"}`
**Response:**
```json
{
  "success": false,
  "message": "Validasi gagal",
  "errors": {
    "Password": "Kata Sandi harus mengandung minimal 8 karakter, huruf besar, huruf kecil, angka, dan simbol"
  },
  "details": [
    {
      "field": "Password",
      "message": "Kata Sandi harus mengandung minimal 8 karakter, huruf besar, huruf kecil, angka, dan simbol"
    }
  ]
}
```

## Advanced Usage

### Nested Validation dengan Label
```go
type CreatePostRequest struct {
    Title   string   `json:"title" label:"Judul" binding:"required,min=5,max=100"`
    Content string   `json:"content" label:"Konten" binding:"required,min=10"`
    Tags    []string `json:"tags" label:"Tag" binding:"required,min=1,max=10,dive,required,no_space,alphanum"`
    Author  Author   `json:"author" label:"Penulis" binding:"required"`
}

type Author struct {
    Name  string `json:"name" label:"Nama Penulis" binding:"required,min=2,max=50"`
    Email string `json:"email" label:"Email Penulis" binding:"required,email"`
}
```

### Multiple Error Handling
```go
func HandleComplexValidation(c *gin.Context) {
    var req CreatePostRequest
    
    if err := c.ShouldBindJSON(&req); err != nil {
        // Akan menangani semua error sekaligus
        utils.SendValidationError(c, err, &req)
        return
    }
    
    // Additional business logic validation
    if req.Title == "forbidden" {
        c.JSON(400, gin.H{
            "success": false,
            "message": "Judul tidak diperbolehkan",
            "errors": map[string]string{
                "title": "Judul 'forbidden' tidak diperbolehkan",
            },
        })
        return
    }
    
    // Process valid request...
}
```

## Best Practices

### 1. **Gunakan Label yang Descriptive**
```go
// ✅ Good
Username string `json:"username" label:"Nama Pengguna" binding:"required,username"`

// ❌ Bad
Username string `json:"username" binding:"required,username"`
```

### 2. **Kombinasi Validator yang Tepat**
```go
// ✅ Good - Array dengan validasi elemen
Codes []string `json:"codes" label:"Kode" binding:"required,min=1,unique,dive,required,no_space,alphanum"`

// ❌ Bad - Tidak ada validasi elemen
Codes []string `json:"codes" binding:"required"`
```

### 3. **Error Handling yang Konsisten**
```go
// ✅ Good
if err := c.ShouldBindJSON(&req); err != nil {
    utils.SendValidationError(c, err, &req)
    return
}

// ❌ Bad - Tidak menggunakan utility
if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(400, gin.H{"error": err.Error()})
    return
}
```

## Testing

Gunakan file `examples/validation_example.go` untuk testing berbagai skenario validasi:

```bash
cd /path/to/your/project
go run examples/validation_example.go
```

## Dependencies

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/gin-gonic/gin/binding"
    "github.com/go-playground/validator/v10"
)
```

Sistem validasi ini memberikan foundation yang solid untuk handling error dengan user experience yang baik melalui pesan error yang jelas dan terstruktur dalam Bahasa Indonesia.
