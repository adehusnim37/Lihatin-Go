# Request validation

Package ini menyediakan custom rules, pesan validasi Indonesia, dan adapter respons Gin.
Registrasikan aturan satu kali saat startup, lalu gunakan kembali instance-nya.

## Struct dengan tag `validate`

```go
import appvalidator "github.com/adehusnim37/lihatin-go/internal/pkg/validator"

validate, err := appvalidator.New()
if err != nil {
    return err
}

req := struct {
    Username string `validate:"required,username" label:"Username"`
}{Username: "nama_pengguna"}
if err := validate.Struct(req); err != nil {
    response := appvalidator.FormatError(err, &req)
    // response.ErrorMap berisi pesan per field.
    _ = response
}
```

## Gin dengan tag `binding`

Gin memakai validator sendiri dan membaca tag `binding`. Daftarkan aturan pada
instance tersebut sebelum server menerima request; `Register` mempertahankan tag
name yang sudah dikonfigurasi. Contoh integrasinya ada di `main.go`.

```go
engine, ok := binding.Validator.Engine().(*validator.Validate)
if !ok {
    return errors.New("unsupported Gin validator engine")
}
if err := appvalidator.Register(engine); err != nil {
    return err
}
```

Di handler, teruskan error dan request asli agar pesan memakai tag `label`, lalu
nama `json` sebagai fallback. Kunci error tetap nama field Go untuk kompatibilitas
API yang ada. Label otomatis berlaku untuk field pada struct yang diteruskan;
field bersarang yang tidak ditemukan memakai nama field dari error.

```go
var req struct {
    Username string `json:"username" binding:"required,username" label:"Username"`
}
if err := ctx.ShouldBindJSON(&req); err != nil {
    appvalidator.SendValidationError(ctx, err, &req)
    return
}
```

Respons HTTP tetap memakai `success`, `data`, `message`, dan `error`.
`FormatError` juga mengenali JSON/validation errors yang dibungkus menggunakan `%w`.
`SetupCustomValidators` dipertahankan untuk kompatibilitas; gunakan `New` atau
`Register` pada kode baru. Jangan melakukan registrasi ulang di setiap handler.

## Aturan yang tersedia

| Tag | Ketentuan |
| --- | --- |
| `pwdcomplex` | Minimal 8 byte, huruf besar/kecil ASCII, angka, dan simbol yang didukung |
| `username`, `saveurlshort` | Huruf ASCII, angka, `_`, dan `-` |
| `no_space` | Tidak mengandung spasi ASCII (` `) |
| `no_special` | Huruf ASCII, angka, atau whitespace yang didukung regexp Go (`\s`) |
| `six_digit` | Integer antara 100000 dan 999999 |
| `secret_code` | 20–80 karakter: huruf ASCII, angka, `-`, atau whitespace `\s` |
| `not_same_digit` | String tidak terdiri dari satu karakter berulang; input kosong lolos |
| `meaningful_text` | Menolak pola gibberish berdasarkan heuristik; input kosong lolos |
| `unique` | Aturan bawaan go-playground untuk nilai unik pada collection |

Tag string harus dipasang pada field string, dan `six_digit` pada signed integer.
Gunakan `required`, `omitempty`, `min`, `max`, serta aturan bawaan lainnya untuk
menentukan apakah field wajib diisi dan batas ukurannya.

## Meaningful text

Rule ini aktif pada subjek/deskripsi tiket support, judul/deskripsi shortlink
(create/update, termasuk single dan bulk), serta pesan support dari pengunjung
dan pengguna login. Pesan multipart diperiksa sebelum upload lampiran dan
penyimpanan pesan. Pesan tanpa body tetap memerlukan lampiran.

Untuk field teks lain, tambahkan tag secara eksplisit:

```go
Body string `binding:"required,min=10,max=5000,meaningful_text"`
```

Untuk multipart atau string langsung, panggil
`appvalidator.IsMeaningfulText(body)` setelah memeriksa requiredness dan panjang.
String kosong tetap lolos agar dapat digabung dengan `omitempty` atau alur
lampiran. Input berisi whitespace saja dan UTF-8 tidak valid ditolak.

Gunakan API analisis untuk mendapatkan alasan tanpa mengubah teks asli:

```go
result := appvalidator.AnalyzeMeaningfulText(body)
switch result.Classification {
case appvalidator.TextGibberish:
    // Tolak input dengan pola yang mendominasi teks.
case appvalidator.TextSuspicious:
    // Tetap terima; sinyal belum cukup untuk menolak seluruh input.
case appvalidator.TextValid:
    // Tidak ada sinyal penolakan yang kuat.
}
// result.Reasons: repetition, keyboard_pattern, consonant_run, dan lainnya.
// result.Score adalah bobot heuristik, bukan probabilitas kebermaknaan.
```

`IsMeaningfulText` menerima `valid` dan `suspicious`; tag `meaningful_text`
memakai keputusan yang sama. `analyzeMeaningfulText` tetap helper internal untuk
menghitung metrik. Pada DTO, pemeriksaan panjang berjalan sebelum analisis teks.

Analisis memakai salinan NFC dan mempertahankan combining marks pada token.
Pengulangan karakter/potongan kata diperiksa lintas aksara; pengulangan frasa
mempertimbangkan hingga empat kata. Keyboard mash diperiksa per token memakai
arah maju/mundur pada baris dan kolom keyboard QWERTY.

Pola keyboard atau repetisi harus mencakup minimal 60% huruf prosa untuk
menolak pesan. Keyboard mash juga memerlukan minimal delapan huruf prosa;
token pendek yang ambigu tetap `suspicious`. Repetisi dalam satu token dinilai
jika mencakup minimal delapan rune dan 50% token tersebut. Kombinasi konsonan
panjang yang mendominasi dan rasio vokal rendah juga dapat menolak teks.
Ambang ini merupakan kebijakan awal, perlu dievaluasi dengan input nyata.

URL, path, UUID, hash heksadesimal panjang, serta kode terstruktur seperti
`ERR_CONNECTION_REFUSED` dikecualikan per token. Teks di sekitarnya tetap
diperiksa. Penyebutan `qwerty` dalam laporan normal tidak otomatis menolak pesan.

Lingua menjadi sinyal tambahan untuk singkatan pendek Indonesia/Inggris dan
baru diinisialisasi saat diperlukan. Confidence Lingua membandingkan bahasa,
bukan memastikan teks bermakna. Deteksi ini belum memakai model Markov/ngram
terlatih; teks acak yang menyerupai kata normal masih dapat lolos. Evaluasi
contoh yang diterima/ditolak per field untuk mengkalibrasi kebijakan ini.

## Pengujian

```sh
go test ./internal/pkg/validator ./dto ./controllers/support ./controllers/shortlink
go test -race ./internal/pkg/validator ./dto ./controllers/support
```
