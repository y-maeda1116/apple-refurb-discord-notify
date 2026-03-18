package main

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestSendDiscordEmbed(t *testing.T) {
	// 環境変数が設定されているか確認
	if os.Getenv("DISCORD_WEBHOOK_URL") == "" {
		t.Skip("DISCORD_WEBHOOK_URL が設定されていないためスキップします")
	}

	// テスト用のダミー商品データ
	testProduct := NormalizedProduct{
		Name:     "テスト用 Mac mini (通知テスト)",
		Price:    "¥100,000",
		PriceRaw: 100000,
		URL:      "https://www.apple.com/jp/shop/refurbished/mac/mac-mini",
		RAM:      "24GB",
		RAMGB:    24,
		Chip:     "M4",
	}

	// 通知関数を実行
	success := sendDiscordEmbed(testProduct, "new")

	if !success {
		t.Error("Discordへの通知送信に失敗しました。Webhook URLを確認してください。")
	}
}

func TestFetchAndNotifyMacBookAir(t *testing.T) {
	// Webhook URLのチェック
	if os.Getenv("DISCORD_WEBHOOK_URL") == "" {
		t.Skip("DISCORD_WEBHOOK_URL が設定されていないためスキップします")
	}

	// 1. 実際にAppleから製品一覧を取得
	products, err := fetchProducts()
	if err != nil {
		t.Fatalf("製品の取得に失敗しました: %v", err)
	}

	var targetProduct *Product
	for _, p := range products {
		// MacBook Airを探す（モデル名に "air" が含まれるか確認）
		if strings.Contains(strings.ToLower(p.Dimensions.RefurbClearModel), "air") {
			targetProduct = &p
			break // 1つ見つかればOK
		}
	}

	if targetProduct == nil {
		t.Skip("現在、整備済製品に MacBook Air の在庫がないためテストをスキップします")
	}

	// 2. 取得したデータを正規化（テスト用にフィルタ条件を無視）
	ramGB := extractRAM(*targetProduct)

	// 価格のパース（main.goのロジックを流用）
	priceStr := strings.TrimSpace(targetProduct.ProductTile.Price.CurrentPrice)
	priceStr = strings.ReplaceAll(priceStr, "¥", "")
	priceStr = strings.ReplaceAll(priceStr, ",", "")
	priceStr = strings.ReplaceAll(priceStr, "JPY", "")
	priceRaw, _ := strconv.Atoi(priceStr)

	normalized := NormalizedProduct{
		Name:     "【テスト】" + targetProduct.Dimensions.RefurbClearModel,
		Price:    targetProduct.ProductTile.Price.CurrentPrice,
		PriceRaw: priceRaw,
		URL:      appleURL + "?fproduct=" + targetProduct.ProductTile.ID,
		RAM:      strconv.Itoa(ramGB) + "GB",
		RAMGB:    ramGB,
		Chip:     "Test",
	}

	// 3. Discordに送信
	t.Logf("テスト送信中: %s", normalized.Name)
	success := sendDiscordEmbed(normalized, "new")

	if !success {
		t.Error("Discordへの送信に失敗しました")
	}
}
