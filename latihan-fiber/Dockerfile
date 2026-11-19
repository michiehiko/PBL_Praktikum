# syntax=docker/dockerfile:1

# =========================================================================
# Tahap 1 — build: image besar berisi Go compiler, HANYA untuk compile.
# =========================================================================
FROM golang:1.26-alpine AS build
WORKDIR /src

# go.mod dan go.sum disalin lebih dulu, terpisah dari kode. Docker menyimpan
# hasil setiap instruksi sebagai layer. Selama dua file ini tidak berubah,
# layer "go mod download" dipakai ulang dari cache dan dependency tidak
# diunduh lagi — walaupun kode program berubah.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Proyek ini punya DUA entrypoint (bukan satu binary dengan flag -role):
#   .            -> API
#   ./cmd/worker -> worker outbox/notifikasi
# Keduanya dibangun di sini, lalu masing-masing disalin ke target runtime-nya.
#
# CGO_ENABLED=0 menghasilkan binary statis yang tidak bergantung pada
# library C, sehingga dapat berjalan di image runtime yang sangat kecil.
# -trimpath membuang path folder komputer pembuat dari binary.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker

# =========================================================================
# Tahap 2 — runtime: hanya binary, tanpa compiler dan tanpa source code.
# =========================================================================
FROM alpine:3.24 AS runtime

# Aplikasi TIDAK dijalankan sebagai root. Bila suatu saat ada celah
# keamanan di aplikasi, penyerang hanya mendapat hak user biasa.
RUN addgroup -S app && adduser -S -G app app

WORKDIR /app

# Logger menulis ke logs/app.log dan logs/mail.log, maka foldernya harus
# dapat ditulis user app.
RUN mkdir -p logs && chown app:app logs
USER app

EXPOSE 3000

# -------------------------------------------------------------------------
# Target WORKER. Binary tetap diberi nama "app" di dalam image agar isi
# /app konsisten dengan target api (hanya berisi "app" dan "logs").
# -------------------------------------------------------------------------
FROM runtime AS worker
COPY --from=build /out/worker ./app
ENTRYPOINT ["./app"]

# -------------------------------------------------------------------------
# Target API. Diletakkan terakhir agar `docker build -t latihan-fiber:local .`
# tanpa --target menghasilkan image API.
# -------------------------------------------------------------------------
FROM runtime AS api
COPY --from=build /out/api ./app
ENTRYPOINT ["./app"]
