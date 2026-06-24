# saisei-file-server（Render版）ver.1.5.0

ファイルのアップロード・管理・課題提出ができるWebサーバー。  
Render + Supabase Storage を使って**完全無料**で外部公開できる。

---

## 機能

1. ログイン（DB認証）
2. ユーザ追加（先生のみ）
3. パスワード変更（全員）
4. フォルダアップロード → zip化
5. アップロード者・日時の記録
6. ファイル一覧表示
7. ダウンロード
8. 削除（先生のみ）
9. 実行（先生のみ）← project.toml 対応
10. 課題作成・一覧（先生・admin作成、全員閲覧）
11. 課題への提出紐づけ（期限付き・複数提出対応）
12. 遅延提出通知（先生のみ）
13. 提出状況モーダル
14. 複数フォルダアップロード
15. ファイル検索
16. リロード時のログイン状態保持
17. 古い提出ファイルの自動削除（ディスク節約）

---

## 構成

```
.
├── render.yaml           # Render Blueprint（サービス定義）
├── server/
│   ├── main.go           # Goサーバー本体
│   ├── storage.go        # Supabase Storage対応（追加）
│   ├── db.go             # DB操作
│   ├── run_helper.go     # 実行ヘルパー
│   ├── pyrunner.py       # project.toml ベース実行エンジン
│   ├── go.mod
│   ├── Dockerfile
│   ├── static/           # フロントエンド
│   └── data/             # ユーザーDB・アップロード履歴
└── task-server/
    ├── main.py           # Python/FastAPI 課題管理サーバー
    ├── requirements.txt
    └── Dockerfile
```

---

## デプロイ手順（Render + Supabase）

### 1. Supabase でストレージを用意する

1. https://supabase.com にGitHubでログイン（クレカ不要）
2. **New Project** を作成
3. 左メニュー → **Storage** → **New Bucket**
   - 名前: `uploads`
   - Public: オフ（非公開）
4. **Project Settings → API** から以下をコピーしておく
   - **Project URL**（例: `https://xxxx.supabase.co`）
   - **service_role** キー（secretの方）

### 2. GitHubにリポジトリを作成してプッシュ

```bash
git init
git add .
git commit -m "initial commit"
git remote add origin git@github-xxxx:YOUR_USER/YOUR_REPO.git
git push -u origin main
```

> SSHのホスト名エイリアスを使っている場合は `git@github.com` ではなく `~/.ssh/config` に合わせたホスト名にする。

### 3. Render でサービスを2つ作成する

`render.yaml` があるが、**Blueprint では Dockerfile Path が自動設定されないことがある**ので、**New+ → Web Service** で手動で2つ作るのが確実。

**① saisei-task-server**

| 項目 | 値 |
|---|---|
| Runtime | Docker |
| Dockerfile Path | `./task-server/Dockerfile` |
| Docker Context | `./task-server` |

**② saisei-file-server**

| 項目 | 値 |
|---|---|
| Runtime | Docker |
| Dockerfile Path | `./server/Dockerfile` |
| Docker Context | `./server` |

> **ポイント**: Dockerfile Path と Docker Context は Settings から手動で設定する。デフォルトの `./Dockerfile` のままだとエラーになる。

### 4. 環境変数を設定する

**saisei-file-server** の Environment Variables に以下を追加：

| Key | Value |
|---|---|
| `SUPABASE_URL` | Supabase の Project URL |
| `SUPABASE_SERVICE_KEY` | Supabase の service_role キー |
| `SUPABASE_BUCKET` | `uploads` |
| `TASK_SERVER_URL` | saisei-task-server の URL（例: `https://xxxx.onrender.com`） |

Save Changes → 自動リデプロイ。

### 5. 動作確認

- `https://saisei-file-server.onrender.com` にアクセス
- デフォルトアカウント: `admin` / `admin123`
- アップロードしたファイルは Supabase Storage の `uploads` バケットに保存される

---

## ハマりポイントと解決策

| 問題 | 原因 | 解決策 |
|---|---|---|
| `open Dockerfile: no such file or directory` | Dockerfile Path がデフォルト（`./Dockerfile`）のまま | Settings で `./server/Dockerfile` に変更 |
| `main.py not found` | Docker Context がリポジトリルートになっている | Settings で `./task-server` に変更 |
| `remote origin already exists` | git remote がすでに設定済み | `git remote remove origin` してから再追加 |
| `Repository not found` | SSH のホスト名エイリアスと remote URL が不一致 | `~/.ssh/config` のホスト名に合わせて remote を設定 |
| ファイルがリデプロイで消える | Render 無料プランはディスク非永続 | Supabase Storage に保存するよう対応済み |

---

## 環境変数一覧

| 変数名 | 説明 | デフォルト |
|---|---|---|
| `PORT` | Render が自動設定するポート番号 | 4450 |
| `TZ` | タイムゾーン | Asia/Tokyo |
| `TASK_SERVER_URL` | task-server の URL | http://task-server:8000 |
| `RUN_TIMEOUT_SEC` | 実行タイムアウト（秒） | 60 |
| `SUPABASE_URL` | Supabase Project URL | （必須） |
| `SUPABASE_SERVICE_KEY` | Supabase service_role キー | （必須） |
| `SUPABASE_BUCKET` | Supabase バケット名 | uploads |

---

## 注意事項

- **スリープ**: 無料プランは15分無操作でスリープ。初回アクセスに30〜60秒かかる
- **「実行」機能**: Render では Docker ソケットが使えないため Python 直接実行のみ動作する
- **task-server のDB**: Render の無料プランではディスク非永続のため、再デプロイでユーザーDBがリセットされる。気になる場合は有料プラン（$7/月〜）または外部DB（Supabase PostgreSQL など）に移行する

---

## デフォルトアカウント

| username | password | role |
|---|---|---|
| admin | admin123 | teacher |
