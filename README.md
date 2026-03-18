# 🍎 Apple Refurbished Mac mini Monitor (Go Version)

Apple公式ストアの**整備済製品（Mac mini）**を15分ごとにチェックし、特定の条件（例: RAM 24GB以上）を満たす出品があった場合に **Discord** へ即座に通知する自動監視ツールです。

## ✨ 特徴
- **超高速・高効率**: Go言語によるバイナリ実行で、1回のチェックを約0.3秒で完了。
- **インテリジェントな通知**: 
  - 新着商品の追加時にDiscordへリッチな埋め込みメッセージを送信。
  - すでに通知済みの商品は `inventory.json` で管理し、重複通知を防止。
- **エコ設計**: GitHub Actionsの無料枠を節約するため、深夜（日本時間 0:00〜7:00）は自動停止。
- **GitHub Actions 連携**: サーバーレスで24時間（稼働時間内）自動稼働。

## 🚀 セットアップ

### 1. Discord Webhook の準備
1. 通知を受け取りたいDiscordチャンネルの設定から「連携サービス」→「ウェブフック」を作成。
2. ウェブフックURLをコピーします。

### 2. GitHub Secrets の設定
GitHubリポジトリの `Settings > Secrets and variables > Actions` に以下の名前でSecretを登録してください。
- `DISCORD_WEBHOOK_URL`: コピーしたDiscordのWebhook URL

### 3. 動作確認（テスト）
ローカル環境で以下のコマンドを実行し、実際にDiscordに通知が飛ぶかテストできます。
```bash
cd go_version
export DISCORD_WEBHOOK_URL="あなたのURL"
# MacBook Airの在庫を1件取得して通知するテスト
go test -v -run TestFetchAndNotifyMacBookAir
