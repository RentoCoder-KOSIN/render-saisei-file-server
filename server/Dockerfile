# ── ステージ1: ビルド ─────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /app

# go.mod をコピー（外部依存ゼロなのでgo.sumは空でOK）
COPY go.mod .
RUN touch go.sum

# ソースをコピーしてビルド
COPY *.go pyrunner.py ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o fileserver .

# ── ステージ2: 実行イメージ ───────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/fileserver .
COPY static/ ./static/
COPY data/ ./data/

# アップロード・データ用ディレクトリ作成
RUN mkdir -p /app/uploads /app/data

EXPOSE 4450
CMD ["./fileserver"]
