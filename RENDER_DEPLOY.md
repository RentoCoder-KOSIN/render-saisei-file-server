# Render へのデプロイ手順

## 構成

| サービス | 技術 | ポート |
|---|---|---|
| `saisei-task-server` | Python / FastAPI | 8000 |
| `saisei-file-server` | Go | 4450 (Render では PORT 環境変数で自動割り当て) |

---

## 手順

### 1. GitHubにリポジトリを作成してプッシュ

```bash
git init
git add .
git commit -m "initial commit"
git remote add origin https://github.com/YOUR_USER/YOUR_REPO.git
git push -u origin main
```

### 2. Render でサービスを作成（Blueprint 利用）

1. https://dashboard.render.com/ にログイン
2. 右上の **New +** → **Blueprint** をクリック
3. GitHubリポジトリを接続
4. `render.yaml` が自動検出されるので **Apply** をクリック

### 3. task-server の URL を file-server に設定

Blueprint 適用後:

1. `saisei-task-server` サービスの URL をコピー  
   例: `https://saisei-task-server.onrender.com`
2. `saisei-file-server` サービス → **Environment** タブを開く
3. `TASK_SERVER_URL` の値をコピーした URL に更新
4. **Save Changes** → 自動リデプロイ

### 4. 動作確認

- `https://saisei-file-server.onrender.com` にアクセス
---

## 注意事項

### 無料プランの制限
- **スリープ**: 15分間アクセスがないとスリープします（初回アクセス時に30〜60秒かかります）
- **ディスク**: 無料プランではディスクが使えません。ファイルの永続化が必要な場合は有料プランが必要です

### 「実行」機能について
ローカル環境では Docker-in-Docker で Python を実行していましたが、Render では Docker ソケットが使えないため、Python が実行環境に存在しない場合はフォールバック動作（実行不可）になります。

### データの永続化
`render.yaml` では Disk を設定していますが、無料プランでは利用できません。  
有料プラン（$7/月〜）にアップグレードするか、PostgreSQL などの外部DBに移行することを推奨します。

---

## 環境変数一覧

| 変数名 | 説明 | デフォルト |
|---|---|---|
| `PORT` | Render が自動設定するポート番号 | 4450 |
| `TZ` | タイムゾーン | Asia/Tokyo |
| `TASK_SERVER_URL` | task-server の URL | http://task-server:8000 |
| `RUN_TIMEOUT_SEC` | 実行タイムアウト（秒） | 60 |
