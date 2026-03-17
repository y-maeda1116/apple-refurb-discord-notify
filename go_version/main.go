package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	appleURL          = "https://www.apple.com/jp/shop/refurbished/mac/mac-mini"
	inventoryFilePath = "../inventory.json"
	discordColor      = 5814783 // Apple Green
)

type Product struct {
	Dimensions Dimensions `json:"dimensions"`
	ProductTile ProductTile `json:"productTile"`
}

type Dimensions struct {
	RefurbClearModel string `json:"refurbClearModel"`
	DimensionRelYear string `json:"dimensionRelYear"`
	TSMemorySize     string `json:"tsMemorySize"`
}

type ProductTile struct {
	ID    string `json:"id"`
	Price Price   `json:"price"`
}

type Price struct {
	CurrentPrice string `json:"currentPrice"`
}

type NormalizedProduct struct {
	Name     string `json:"name"`
	Price    string `json:"price"`
	PriceRaw int    `json:"price_raw"`
	URL      string `json:"url"`
	RAM      string `json:"ram"`
	RAMGB    int    `json:"ram_gb"`
	Chip     string `json:"chip"`
	Thumbnail string `json:"thumbnail"`
}

type Inventory struct {
	Products   map[string]ProductInfo `json:"products"`
	LastFetch  string               `json:"last_fetch"`
}

type ProductInfo struct {
	Name          string `json:"name"`
	RAM           string `json:"ram"`
	RAMGB         int    `json:"ram_gb"`
	Price         string `json:"price"`
	PriceRaw      int    `json:"price_raw"`
	FirstNotified string `json:"first_notified,omitempty"`
	LastNotified  string `json:"last_notified,omitempty"`
	InStock       bool   `json:"in_stock"`
}

type DiscordEmbed struct {
	Embeds []DiscordEmbedData `json:"embeds"`
}

type DiscordEmbedData struct {
	Title     string        `json:"title"`
	URL       string        `json:"url"`
	Color     int           `json:"color"`
	Thumbnail DiscordThumbnail `json:"thumbnail"`
	Fields    []DiscordField `json:"fields"`
	Footer    DiscordFooter `json:"footer"`
}

type DiscordThumbnail struct {
	URL string `json:"url"`
}

type DiscordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type DiscordFooter struct {
	Text string `json:"text"`
}

var logger = log.New(os.Stdout, "", log.LstdFlags)

func fetchProducts() ([]Product, error) {
	logger.Printf("Fetching products from %s", appleURL)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(appleURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch products: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Extract products array from HTML
	html := string(body)
	re := regexp.MustCompile(`products"\s*:\s*(\[[^\]]+(?:\[[^\]]*\][^\]]*)*\])`)
	matches := re.FindStringSubmatch(html)
	if len(matches) < 2 {
		logger.Println("Could not find products data in HTML")
		return []Product{}, nil
	}

	// Fix JSON syntax (JavaScript to Go)
	productsJSON := matches[1]
	productsJSON = regexp.MustCompile(`(\w+)\s*:`).ReplaceAllString(productsJSON, `"$1":`)
	productsJSON = strings.ReplaceAll(productsJSON, "'", "\"")

	var products []Product
	if err := json.Unmarshal([]byte(productsJSON), &products); err != nil {
		return nil, fmt.Errorf("failed to parse products JSON: %w", err)
	}

	logger.Printf("Fetched %d products from Apple", len(products))
	return products, nil
}

func extractRAM(product Product) int {
	// Try tsMemorySize first
	ramStr := strings.ToLower(product.Dimensions.TSMemorySize)
	if ramStr != "" {
		re := regexp.MustCompile(`(\d+)`)
		matches := re.FindStringSubmatch(ramStr)
		if len(matches) > 1 {
			ram, _ := strconv.Atoi(matches[1])
			return ram
		}
	}
	return 0
}

func filterProducts(products []Product) []NormalizedProduct {
	var filtered []NormalizedProduct

	for _, product := range products {
		model := strings.ToLower(product.Dimensions.RefurbClearModel)
		if !strings.Contains(model, "mini") {
			continue
		}

		ramGB := extractRAM(product)
		if ramGB < 24 {
			continue
		}

		// Build URL
		productID := product.ProductTile.ID
		var url string
		if productID != "" {
			url = fmt.Sprintf("%s?fproduct=%s", appleURL, productID)
		} else {
			url = appleURL
		}

		// Parse price
		priceStr := strings.TrimSpace(product.ProductTile.Price.CurrentPrice)
		priceStr = strings.ReplaceAll(priceStr, "¥", "")
		priceStr = strings.ReplaceAll(priceStr, ",", "")
		priceStr = strings.ReplaceAll(priceStr, "JPY", "")
		priceStr = strings.ReplaceAll(priceStr, ".", "")

		priceRaw, err := strconv.Atoi(priceStr)
		if err != nil || priceRaw == 0 {
			continue
		}

		// Extract chip name
		year := product.Dimensions.DimensionRelYear
		chip := "Unknown"
		if len(year) >= 2 {
			chip = "M" + year[len(year)-2:]
		}

		normalized := NormalizedProduct{
			Name:     "Mac mini " + chip,
			Price:    fmt.Sprintf("¥%d,", priceRaw),
			PriceRaw: priceRaw,
			URL:      url,
			RAM:      fmt.Sprintf("%dGB", ramGB),
			RAMGB:    ramGB,
			Chip:     chip,
		}

		filtered = append(filtered, normalized)
		logger.Printf("Matched: %s - %s - %s", normalized.Name, normalized.RAM, normalized.Price)
	}

	logger.Printf("Filtered to %d Mac mini products with 24GB+ RAM", len(filtered))
	return filtered
}

func generateUniqueID(product NormalizedProduct) string {
	key := fmt.Sprintf("%s_%d_%d", product.Name, product.RAMGB, product.PriceRaw)
	hash := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", hash)[:16]
}

func loadInventory() (*Inventory, error) {
	inventory := &Inventory{
		Products: make(map[string]ProductInfo),
	}

	data, err := os.ReadFile(inventoryFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Println("Inventory file not found, creating new")
			return inventory, nil
		}
		return nil, fmt.Errorf("failed to read inventory: %w", err)
	}

	if err := json.Unmarshal(data, inventory); err != nil {
		logger.Printf("Failed to parse inventory.json: %v, creating new", err)
		inventory.Products = make(map[string]ProductInfo)
		return inventory, nil
	}

	logger.Printf("Loaded inventory with %d products", len(inventory.Products))
	return inventory, nil
}

func saveInventory(inventory *Inventory) error {
	inventory.LastFetch = time.Now().UTC().Format(time.RFC3339)
	data, err := json.MarshalIndent(inventory, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal inventory: %w", err)
	}

	if err := os.WriteFile(inventoryFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write inventory: %w", err)
	}

	logger.Println("Saved inventory.json")
	return nil
}

func sendDiscordEmbed(product NormalizedProduct, status string) bool {
	webhookURL := os.Getenv("DISCORD_WEBHOOK_URL")
	if webhookURL == "" {
		logger.Println("DISCORD_WEBHOOK_URL not set, skipping notification")
		return false
	}

	statusText := "新着 🆕"
	if status != "new" {
		statusText = "再入庫 🆕"
	}

	priceFormatted := fmt.Sprintf("¥%d,", product.PriceRaw)
	priceFormatted = priceFormatted[:len(priceFormatted)-1] + ","

	embed := DiscordEmbed{
		Embeds: []DiscordEmbedData{
			{
				Title: "🍎 整備済製品: " + product.Name,
				URL:   product.URL,
				Color: discordColor,
				Thumbnail: DiscordThumbnail{
					URL: product.Thumbnail,
				},
				Fields: []DiscordField{
					{Name: "💰 価格", Value: fmt.Sprintf("¥%d,", product.PriceRaw), Inline: true},
					{Name: "💾 RAM", Value: product.RAM, Inline: true},
					{Name: "🔁 状態", Value: statusText, Inline: true},
				},
				Footer: DiscordFooter{
					Text: "Apple Refurbished Monitor | " + time.Now().UTC().Format("2006-01-02 15:04"),
				},
			},
		},
	}

	data, _ := json.Marshal(embed)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(webhookURL, "application/json", strings.NewReader(string(data)))
	if err != nil {
		logger.Printf("Failed to send Discord notification: %v", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Printf("Discord webhook returned status: %d", resp.StatusCode)
		return false
	}

	logger.Printf("Sent Discord notification for: %s", product.Name)
	return true
}

func commitInventory() bool {
	// Check if there are changes
	cmd := exec.Command("git", "diff", "--quiet", inventoryFilePath)
	cmd.Dir = ".."
	if err := cmd.Run(); err == nil {
		logger.Println("No changes to inventory.json, skipping commit")
		return false
	}

	// Add and commit changes
	exec.Command("git", "add", inventoryFilePath).Run()
	cmd = exec.Command("git", "commit", "-m", "chore: update inventory [skip ci]")
	cmd.Dir = ".."
	if err := cmd.Run(); err != nil {
		logger.Printf("Failed to commit inventory.json: %v", err)
		return false
	}
	logger.Println("Committed inventory.json changes")

	// Push changes
	cmd = exec.Command("git", "push")
	cmd.Dir = ".."
	if err := cmd.Run(); err != nil {
		logger.Printf("Failed to push: %v", err)
		return false
	}
	logger.Println("Pushed inventory.json changes")
	return true
}

func main() {
	if os.Getenv("DISCORD_WEBHOOK_URL") == "" {
		logger.Println("DISCORD_WEBHOOK_URL environment variable not set")
		os.Exit(1)
	}

	// 1. Fetch products from Apple
	products, err := fetchProducts()
	if err != nil {
		logger.Fatalf("Failed to fetch products: %v", err)
	}

	// 2. Filter for Mac mini with 24GB+ RAM
	filtered := filterProducts(products)
	if len(filtered) == 0 {
		logger.Println("No matching products found")
		return
	}

	// 3. Load inventory
	inventory, err := loadInventory()
	if err != nil {
		logger.Fatalf("Failed to load inventory: %v", err)
	}

	// 4. Process each product
	currentProductIDs := make(map[string]bool)
	notifiedCount := 0

	for _, product := range filtered {
		uniqueID := generateUniqueID(product)
		currentProductIDs[uniqueID] = true

		var status string
		if info, exists := inventory.Products[uniqueID]; !exists {
			status = "new"
			logger.Printf("New product found: %s", product.Name)
		} else if !info.InStock {
			status = "reinstated"
			logger.Printf("Restock detected: %s", product.Name)
		} else {
			// Still in stock, skip
			continue
		}

		// Send notification
		if sendDiscordEmbed(product, status) {
			notifiedCount++

			// Update inventory
			now := time.Now().UTC().Format(time.RFC3339)
			info, exists := inventory.Products[uniqueID]
			if exists {
				info.LastNotified = now
				info.InStock = true
				info.Price = product.Price
				info.PriceRaw = product.PriceRaw
				inventory.Products[uniqueID] = info
			} else {
				inventory.Products[uniqueID] = ProductInfo{
					Name:          product.Name,
					RAM:           product.RAM,
					RAMGB:         product.RAMGB,
					Price:         product.Price,
					PriceRaw:      product.PriceRaw,
					FirstNotified: now,
					LastNotified:  now,
					InStock:       true,
				}
			}
		}
	}

	// Mark products that disappeared as out of stock
	for id, info := range inventory.Products {
		if !currentProductIDs[id] {
			info.InStock = false
			inventory.Products[id] = info
			logger.Printf("Marked out of stock: %s", info.Name)
		}
	}

	// 5. Save inventory
	if err := saveInventory(inventory); err != nil {
		logger.Fatalf("Failed to save inventory: %v", err)
	}

	// 6. Commit changes if needed
	if notifiedCount > 0 {
		commitInventory()
	}

	logger.Printf("Completed. Notified: %d products", notifiedCount)
}
