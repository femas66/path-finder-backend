# Dokumentasi API

## Akses

Jalankan backend seperti biasa dengan PostgreSQL dan `JWT_SECRET` terpasang.
Port default 3000:

- Swagger UI: http://localhost:3000/docs
- OpenAPI 3.0.3 JSON: http://localhost:3000/openapi.json

Dokumentasi dapat diakses tanpa login. Endpoint bisnis terlindungi tetap
memerlukan JWT dan role sesuai dokumentasi.

Login melalui `POST /api/users/login` atau alias `/api/auth/login`, salin
`data.token`, lalu masukkan token saja (tanpa prefiks `Bearer`) ke tombol
**Authorize**. Swagger UI menambahkan header Bearer. Token tidak disimpan lintas
reload (`persistAuthorization: false`). Gunakan akun dan database pengujian:
**Try it out menjalankan request sungguhan, termasuk operasi ubah dan hapus.**

## Sumber kontrak

`internal/apidocs/openapi.json` merupakan sumber kontrak statis, bukan hasil
generator. Kontrak dan halaman UI di-embed ke binary Go; runtime tidak perlu
membaca file dari working directory. Tidak ada dependency Go baru.

Saat menambah atau mengubah API:

1. Perbarui route, request/response model, dan implementasi.
2. Perbarui paths, schemas, validasi, status respons, security, serta alias dalam
   `internal/apidocs/openapi.json`. Setiap operasi wajib memiliki operationId unik.
3. Jalankan test dokumentasi dan unit test:

   ```sh
   go test ./test -run '^TestOpenAPIDocumentation$' -count=1
   go test ./internal/...
   ```

Test dokumentasi tidak memerlukan DB atau secret: test mendaftarkan route tanpa
memanggil handler bisnis, memeriksa JSON dan kecocokan operasi terhadap route
aktif. Test ini tidak menggantikan validator formal OpenAPI atau test kontrak
respons bisnis. Endpoint dokumentasi dan HEAD otomatis tidak dihitung sebagai
operasi bisnis. Perubahan kontrak otomatis masuk binary setelah build ulang.

## CDN dan akses jaringan

Swagger UI memakai `swagger-ui-dist@5.17.14` dari `cdn.jsdelivr.net`. Browser
memerlukan akses internet ke CDN untuk CSS/JavaScript. Script CDN berjalan dalam
origin dokumentasi; gunakan kebijakan deployment yang sesuai untuk dokumentasi
yang mengizinkan token. Untuk deployment offline atau larangan script pihak
ketiga, vendor aset UI terlebih dahulu. `validatorUrl: null` menonaktifkan
validator eksternal Swagger agar kontrak tidak dikirim ke layanan validator.
File `/openapi.json` sendiri tidak memerlukan CDN dan bisa diimpor ke Postman
atau alat OpenAPI lain.

## Batas cakupan

Kontrak mencakup endpoint aktif, termasuk alias auth/perusahaan. Dashboard saat
ini hanya identitas login; lupa password hanya mengirim OTP. Jobs CRUD,
verifikasi OTP, serta reset password belum didokumentasikan karena belum ada
route. Prediksi dan rekomendasi machine learning dikelola di repository service Python terpisah.
