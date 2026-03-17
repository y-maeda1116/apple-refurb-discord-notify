# Apple Refurbished Mac mini Monitor Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a monitoring system that checks Apple's refurbished products every 15 minutes and sends Discord notifications for Mac mini models with 24GB+ RAM.

**Architecture:** Single Python script that fetches Apple API, filters products, compares against inventory state, and sends Discord embeds. Runs on GitHub Actions with 15-minute cron schedule. State tracked in inventory.json committed to git.

**Tech Stack:** Python 3.11+, requests library, GitHub Actions, Discord Webhook API

---

## Chunk 1: Project Setup

## Chunk 1: Project Setup

### Task 1: Create requirements.txt

**Files:**
- Create: `requirements.txt`

- [ ] **Step 1: Write requirements.txt**

```python
requests==2.31.0
pyyaml==6.0.1
```

- [ ] **Step 2: Commit**

```bash
git add requirements.txt
git commit -m "chore: add requirements.txt"
```

---

### Task 2: Create .gitignore

**Files:**
- Create: `.gitignore`

- [ ] **Step 1: Write .gitignore**

```gitignore
# Python
__pycache__/
*.py[cod]
*$py.class
*.so
.Python
venv/
env/
.venv/

# Environment variables
.env
.env.local

# IDE
.vscode/
.idea/

# macOS
.DS_Store
```

- [ ] **Step 2: Commit**

```bash
git add .gitignore
git commit -m "chore: add .gitignore"
```

---

### Task 3: Create .env.example

**Files:**
- Create: `.env.example`

- [ ] **Step 1: Write .env.example**

```bash
# Discord Webhook URL for sending notifications
DISCORD_WEBHOOK_URL=your_webhook_url_here
```

- [ ] **Step 2: Commit**

```bash
git add .env.example
git commit -m "chore: add .env.example"
```

---

## Chunk 2: Core Monitoring Script

### Task 4: Create src/monitor.py - Imports and Logging

**Files:**
- Create: `src/monitor.py`

- [ ] **Step 0: Create src directory**

```bash
mkdir -p src
```

- [ ] **Step 1: Write imports and logging setup**

```python
#!/usr/bin/env python3
"""
Apple Refurbished Mac mini Monitor

Monitors Apple's refurbished products page for Mac mini models with 24GB+ RAM.
Sends Discord notifications when matching products are found.
"""

import hashlib
import json
import logging
import os
import re
import subprocess
from datetime import datetime, timezone
from pathlib import Path

import requests


# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


# Configuration
APPLE_API_URL = "https://www.apple.com/jp/shop/refurbished/ajax/mac/mac-mini"
INVENTORY_FILE = Path("inventory.json")
DISCORD_WEBHOOK_URL = os.environ.get("DISCORD_WEBHOOK_URL")

if not DISCORD_WEBHOOK_URL:
    logger.error("DISCORD_WEBHOOK_URL environment variable not set")
    raise SystemExit(1)
```

- [ ] **Step 2: Run syntax check**

```bash
python3 -m py_compile src/monitor.py
```
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add src/monitor.py
git commit -m "feat: add monitor.py imports and logging setup"
```

---

### Task 5: Add Apple API Fetch Function

**Files:**
- Modify: `src/monitor.py`

- [ ] **Step 1: Add fetch_products function**

```python
def fetch_products() -> list[dict]:
    """Fetch products from Apple's refurbished API.

    Returns:
        List of product dictionaries from Apple API.

    Raises:
        requests.RequestException: If API request fails.
        json.JSONDecodeError: If response is not valid JSON.
    """
    logger.info(f"Fetching products from {APPLE_API_URL}")

    try:
        response = requests.get(APPLE_API_URL, timeout=30)
        response.raise_for_status()
        data = response.json()

        # Apple API response structure varies - parse accordingly
        if isinstance(data, dict) and "products" in data:
            products = data["products"]
        elif isinstance(data, list):
            products = data
        elif isinstance(data, dict) and "content" in data:
            # Some responses have nested content
            products = data["content"].get("productTileData", {}).get("tiles", [])
        else:
            logger.warning(f"Unexpected API response structure: {type(data)}")
            products = []

        logger.info(f"Fetched {len(products)} products from Apple API")
        return products

    except requests.RequestException as e:
        logger.error(f"Failed to fetch products: {e}")
        raise
    except json.JSONDecodeError as e:
        logger.error(f"Failed to parse JSON response: {e}")
        raise
```

- [ ] **Step 2: Run syntax check**

```bash
python3 -m py_compile src/monitor.py
```
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add src/monitor.py
git commit -m "feat: add fetch_products function"
```

---

### Task 6: Add RAM Extraction Function

**Files:**
- Modify: `src/monitor.py`

- [ ] **Step 1: Add extract_ram function**

```python
def extract_ram(product: dict) -> int | None:
    """Extract RAM value in GB from product data.

    Tries multiple patterns:
    - Structured data: product['parts']['dimensionsCapacity']['ram']
    - Text parsing: "24GB 統合メモリ", "32GB", etc.

    Args:
        product: Product dictionary from Apple API.

    Returns:
        RAM value in GB, or None if not found.
    """
    # Try structured data first
    if 'parts' in product:
        try:
            ram_str = str(product['parts']['dimensionsCapacity'].get('ram', ''))
            match = re.search(r'(\d+)', ram_str)
            if match:
                return int(match.group(1))
        except (KeyError, TypeError):
            pass

    # Try other common structured paths
    for path in ['ram', 'memory', 'memoryCapacity', 'dimensionsCapacity.ram']:
        keys = path.split('.')
        value = product
        for key in keys:
            if isinstance(value, dict):
                value = value.get(key)
                if value is None:
                    break
            else:
                break
        if value:
            match = re.search(r'(\d+)', str(value))
            if match:
                return int(match.group(1))

    # Fallback to regex on name, title, or description fields
    patterns = [r'(\d+)GB', r'(\d+) GB', r'(\d+)ギガバイト']
    text_fields = ['name', 'title', 'shortTitle', 'productName', 'description']

    text = ''
    for field in text_fields:
        if field in product:
            text += str(product[field]) + ' '

    for pattern in patterns:
        match = re.search(pattern, text)
        if match:
            return int(match.group(1))

    logger.warning(f"Could not extract RAM from product: {product.get('name', 'unknown')}")
    return None
```

- [ ] **Step 2: Run syntax check**

```bash
python3 -m py_compile src/monitor.py
```
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add src/monitor.py
git commit -m "feat: add extract_ram function"
```

---

### Task 7: Add Product Filtering Function

**Files:**
- Modify: `src/monitor.py`

- [ ] **Step 1: Add filter_products function**

```python
def filter_products(products: list[dict]) -> list[dict]:
    """Filter products for Mac mini with 24GB+ RAM.

    Args:
        products: List of product dictionaries from Apple API.

    Returns:
        List of filtered products as normalized dictionaries.
    """
    filtered = []

    for product in products:
        # Check if product name contains "Mac mini"
        name = product.get('name', product.get('title', product.get('productName', '')))
        if 'Mac mini' not in name and 'MacMini' not in name:
            continue

        # Extract RAM
        ram_gb = extract_ram(product)
        if ram_gb is None or ram_gb < 24:
            continue

        # Normalize product data
        normalized = {
            "name": name,
            "price": product.get('price', {}).get('current', {}).get('price', ''),
            "price_raw": 0,
            "url": f"https://www.apple.com/jp/shop/refurbished{product.get('productUrl', '')}",
            "ram": f"{ram_gb}GB",
            "ram_gb": ram_gb,
            "chip": "Unknown",
            "thumbnail": product.get('image', {}).get('src', '')
        }

        # Parse price to integer
        price_str = normalized["price"].replace('¥', '').replace(',', '').replace('JPY', '').strip()
        try:
            normalized["price_raw"] = int(price_str)
            if normalized["price_raw"] == 0:
                continue  # Invalid price
        except ValueError:
            logger.warning(f"Could not parse price: {price_str}")
            continue

        # Extract chip name if available
        chip = product.get('chip', '')
        if chip:
            normalized["chip"] = chip

        filtered.append(normalized)
        logger.info(f"Matched: {normalized['name']} - {normalized['ram']} - {normalized['price']}")

    logger.info(f"Filtered to {len(filtered)} Mac mini products with 24GB+ RAM")
    return filtered
```

- [ ] **Step 2: Run syntax check**

```bash
python3 -m py_compile src/monitor.py
```
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add src/monitor.py
git commit -m "feat: add filter_products function"
```

---

### Task 8: Add Unique ID Generation Function

**Files:**
- Modify: `src/monitor.py`

- [ ] **Step 1: Add generate_unique_id function**

```python
def generate_unique_id(product: dict) -> str:
    """Generate unique ID from product name, RAM, and price.

    Args:
        product: Normalized product dictionary.

    Returns:
        Unique ID string (16 characters).
    """
    key = f"{product['name']}_{product['ram_gb']}_{product['price_raw']}"
    return hashlib.sha256(key.encode()).hexdigest()[:16]
```

- [ ] **Step 2: Run syntax check**

```bash
python3 -m py_compile src/monitor.py
```
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add src/monitor.py
git commit -m "feat: add generate_unique_id function"
```

---

### Task 9: Add Inventory Load/Save Functions

**Files:**
- Modify: `src/monitor.py`

- [ ] **Step 1: Add load_inventory and save_inventory functions**

```python
def load_inventory() -> dict:
    """Load inventory.json.

    Returns:
        Inventory dictionary. Empty dict if file doesn't exist.
    """
    if not INVENTORY_FILE.exists():
        logger.info("Inventory file not found, creating new")
        return {"products": {}, "last_fetch": None}

    try:
        with open(INVENTORY_FILE, 'r', encoding='utf-8') as f:
            inventory = json.load(f)
        logger.info(f"Loaded inventory with {len(inventory.get('products', {}))} products")
        return inventory
    except json.JSONDecodeError as e:
        logger.error(f"Failed to parse inventory.json: {e}")
        return {"products": {}, "last_fetch": None}


def save_inventory(inventory: dict) -> None:
    """Save inventory.json.

    Args:
        inventory: Inventory dictionary to save.
    """
    inventory["last_fetch"] = datetime.now(timezone.utc).isoformat()
    with open(INVENTORY_FILE, 'w', encoding='utf-8') as f:
        json.dump(inventory, f, ensure_ascii=False, indent=2)
    logger.info("Saved inventory.json")
```

- [ ] **Step 2: Run syntax check**

```bash
python3 -m py_compile src/monitor.py
```
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add src/monitor.py
git commit -m "feat: add load_inventory and save_inventory functions"
```

---

### Task 10: Add Discord Notification Function

**Files:**
- Modify: `src/monitor.py`

- [ ] **Step 1: Add send_discord_embed function**

```python
def send_discord_embed(product: dict, status: str) -> bool:
    """Send Discord embed notification for a product.

    Args:
        product: Normalized product dictionary.
        status: "new" or "reinstated".

    Returns:
        True if notification was sent successfully, False otherwise.
    """
    # Format price with 3-digit grouping
    price_formatted = f"¥{product['price_raw']:,}"

    # Status display text
    if status == "new":
        status_text = "新着 🆕"
    else:
        status_text = "再入庫 🆕"

    # Discord embed color (Apple Green: #589632)
    color = 5814783

    embed = {
        "embeds": [{
            "title": f"🍎 整備済製品: {product['name']}",
            "url": product["url"],
            "color": color,
            "thumbnail": {
                "url": product.get("thumbnail", "")
            },
            "fields": [
                {
                    "name": "💰 価格",
                    "value": price_formatted,
                    "inline": True
                },
                {
                    "name": "💾 RAM",
                    "value": product["ram"],
                    "inline": True
                },
                {
                    "name": "🔁 状態",
                    "value": status_text,
                    "inline": True
                }
            ],
            "footer": {
                "text": f"Apple Refurbished Monitor | {datetime.now(timezone.utc).strftime('%Y-%m-%d %H:%M')}"
            }
        }]
    }

    try:
        response = requests.post(DISCORD_WEBHOOK_URL, json=embed, timeout=10)
        response.raise_for_status()
        logger.info(f"Sent Discord notification for: {product['name']}")
        return True
    except requests.RequestException as e:
        logger.error(f"Failed to send Discord notification: {e}")
        return False
```

- [ ] **Step 2: Run syntax check**

```bash
python3 -m py_compile src/monitor.py
```
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add src/monitor.py
git commit -m "feat: add send_discord_embed function"
```

---

### Task 11: Add Git Commit Function

**Files:**
- Modify: `src/monitor.py`

- [ ] **Step 1: Add commit_inventory function**

```python
def commit_inventory() -> bool:
    """Commit inventory.json to git if it has changed.

    Returns:
        True if commit was made, False if no changes or error.
    """
    try:
        # Check if there are changes
        result = subprocess.run(
            ["git", "diff", "--quiet", "inventory.json"],
            capture_output=True,
            text=True
        )

        if result.returncode == 0:
            logger.info("No changes to inventory.json, skipping commit")
            return False

        # Stage and commit changes
        subprocess.run(["git", "add", "inventory.json"], check=True)
        subprocess.run(
            ["git", "commit", "-m", "chore: update inventory [skip ci]"],
            check=True
        )
        logger.info("Committed inventory.json changes")

        # Push changes
        subprocess.run(["git", "push"], check=True)
        logger.info("Pushed inventory.json changes")
        return True

    except subprocess.CalledProcessError as e:
        logger.error(f"Failed to commit inventory.json: {e}")
        return False
```

- [ ] **Step 2: Run syntax check**

```bash
python3 -m py_compile src/monitor.py
```
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add src/monitor.py
git commit -m "feat: add commit_inventory function"
```

---

### Task 12: Add Main Function

**Files:**
- Modify: `src/monitor.py`

- [ ] **Step 1: Add main function**

```python
def main() -> int:
    """Main execution function.

    Returns:
        Exit code (0 for success, 1 for error).
    """
    try:
        # 1. Fetch products from Apple API
        products = fetch_products()

        # 2. Filter for Mac mini with 24GB+ RAM
        filtered_products = filter_products(products)

        if not filtered_products:
            logger.info("No matching products found")
            return 0

        # 3. Load inventory
        inventory = load_inventory()

        # 4. Process each product
        current_product_ids = set()
        notified_count = 0

        for product in filtered_products:
            unique_id = generate_unique_id(product)
            current_product_ids.add(unique_id)

            # Check if this is a new product or restock
            if unique_id not in inventory["products"]:
                # New product
                status = "new"
                logger.info(f"New product found: {product['name']}")
            else:
                # Previously seen - check if it was out of stock
                if not inventory["products"][unique_id]["in_stock"]:
                    status = "reinstated"
                    logger.info(f"Restock detected: {product['name']}")
                else:
                    # Still in stock, skip
                    continue

            # Send notification
            if send_discord_embed(product, status):
                notified_count += 1

                # Update inventory
                now = datetime.now(timezone.utc).isoformat()
                if unique_id in inventory["products"]:
                    inventory["products"][unique_id]["last_notified"] = now
                    inventory["products"][unique_id]["in_stock"] = True
                    inventory["products"][unique_id]["price"] = product["price"]
                    inventory["products"][unique_id]["price_raw"] = product["price_raw"]
                else:
                    inventory["products"][unique_id] = {
                        "name": product["name"],
                        "ram": product["ram"],
                        "ram_gb": product["ram_gb"],
                        "price": product["price"],
                        "price_raw": product["price_raw"],
                        "first_notified": now,
                        "last_notified": now,
                        "in_stock": True
                    }

        # Mark products that disappeared as out of stock
        for unique_id in list(inventory["products"].keys()):
            if unique_id not in current_product_ids:
                inventory["products"][unique_id]["in_stock"] = False
                logger.info(f"Marked out of stock: {inventory['products'][unique_id]['name']}")

        # 5. Save inventory
        save_inventory(inventory)

        # 6. Commit changes if needed
        if notified_count > 0:
            commit_inventory()

        logger.info(f"Completed. Notified: {notified_count} products")
        return 0

    except Exception as e:
        logger.error(f"Unexpected error: {e}")
        return 1


if __name__ == "__main__":
    exit(main())
```

- [ ] **Step 2: Run syntax check**

```bash
python3 -m py_compile src/monitor.py
```
Expected: No errors

- [ ] **Step 3: Make script executable**

```bash
chmod +x src/monitor.py
```

- [ ] **Step 4: Commit**

```bash
git add src/monitor.py
git commit -m "feat: add main function with complete monitoring logic"
```

---

## Chunk 3: GitHub Actions Workflow

### Task 13: Create GitHub Actions Workflow

**Files:**
- Create: `.github/workflows/monitor.yml`

- [ ] **Step 0: Create .github/workflows directory**

```bash
mkdir -p .github/workflows
```

- [ ] **Step 1: Write monitor.yml**

```yaml
name: Monitor Apple Refurbished Mac mini

on:
  schedule:
    # Every 15 minutes (UTC)
    - cron: '*/15 * * * *'
  workflow_dispatch:

permissions:
  contents: write

jobs:
  monitor:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout repository
        uses: actions/checkout@v4
        with:
          token: ${{ secrets.GITHUB_TOKEN }}

      - name: Set up Python
        uses: actions/setup-python@v5
        with:
          python-version: '3.11'

      - name: Install dependencies
        run: |
          python -m pip install --upgrade pip
          pip install -r requirements.txt

      - name: Run monitor script
        env:
          DISCORD_WEBHOOK_URL: ${{ secrets.DISCORD_WEBHOOK_URL }}
        run: python src/monitor.py

      - name: Commit inventory changes
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"
          git diff --quiet inventory.json || \
          (git add inventory.json && \
           git commit -m "chore: update inventory [skip ci]" && \
           git push)
```

- [ ] **Step 2: Validate YAML syntax**

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/monitor.yml'))"
```
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/monitor.yml
git commit -m "feat: add GitHub Actions workflow for 15-minute monitoring"
```

---

## Chunk 4: Documentation

### Task 14: Create README.md

**Files:**
- Create/Update: `README.md`

- [ ] **Step 1: Write README.md**

```markdown
# Apple Refurbished Mac mini Monitor

Discord通知でApple整備済製品のMac mini（24GB以上）を監視するシステム。

## 機能

- 15分ごとにAppleの整備済製品ページをチェック
- Mac miniの24GB以上のモデルを検出
- Discordに埋め込み形式で通知
- 再入庫検知機能
- 重複通知防止

## セットアップ

### 1. リポジトリを複製

```bash
git clone <repository-url>
cd apple-refurb-discord-notify
```

### 2. Python依存パッケージをインストール

```bash
python -m venv venv
source venv/bin/activate  # Windows: venv\Scripts\activate
pip install -r requirements.txt
```

### 3. 環境変数を設定

`.env`ファイルを作成して、Discord Webhook URLを設定：

```bash
cp .env.example .env
# .envを編集してDISCORD_WEBHOOK_URLを設定
```

### 4. ローカルでテスト

```bash
python src/monitor.py
```

## GitHub Actions設定

### 1. GitHub SecretsにWebhook URLを設定

GitHubリポジトリで以下のSecretを設定：
- `DISCORD_WEBHOOK_URL`: DiscordのWebhook URL

### 2. ワークフローをプッシュ

```bash
git add .
git commit -m "feat: initial setup"
git push
```

### 3. Actionsタブで確認

GitHubリポジトリのActionsタブで、ワークフローが正常に実行されていることを確認。

## ローカル開発

### テスト実行

```bash
# inventory.jsonを削除して新規通知をテスト
rm inventory.json
python src/monitor.py

# 重複通知の防止をテスト
python src/monitor.py
```

## ディレクトリ構造

```
apple-refurb-discord-notify/
├── .github/
│   └── workflows/
│       └── monitor.yml       # GitHub Actions定義
├── src/
│   └── monitor.py           # メイン監視スクリプト
├── inventory.json           # 通知済み商品管理（自動生成）
├── .env.example             # 環境変数テンプレート
├── .gitignore              # Git除外設定
├── requirements.txt         # Python依存パッケージ
└── README.md               # このファイル
```

## 通知例

```
🍎 整備済製品: Mac mini M4 Pro 24GB

💰 価格: ¥138,800
💾 RAM: 24GB
🔁 状態: 新着 🆕
```

## ライセンス

MIT
```

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add comprehensive README"
```

---

## Final Step: Initial Commit

### Task 15: Create initial inventory.json

**Files:**
- Create: `inventory.json`

- [ ] **Step 1: Create initial inventory.json**

```json
{
  "products": {},
  "last_fetch": null
}
```

- [ ] **Step 2: Commit**

```bash
git add inventory.json
git commit -m "chore: add initial inventory.json"
```

---

### Task 16: Local Testing Verification

**Files:**
- None (verification only)

- [ ] **Step 1: Test script execution**

```bash
python3 -m pip install --upgrade pip
pip3 install -r requirements.txt
python3 src/monitor.py
```
Expected: Script runs without errors, logs output visible

- [ ] **Step 2: Verify inventory.json created**

```bash
cat inventory.json
```
Expected: Valid JSON with products and last_fetch fields

- [ ] **Step 3: Commit verification notes**

```bash
git commit -m "chore: local testing verified [skip ci]"
```

---

## Verification Checklist

After completing all tasks:

- [ ] All tests pass locally
- [ ] Script runs without errors when DISCORD_WEBHOOK_URL is set
- [ ] GitHub Actions workflow is valid YAML
- [ ] README.md is comprehensive
- [ ] All commits follow conventional commit format

## Deployment Steps

1. Push to GitHub repository
2. Set `DISCORD_WEBHOOK_URL` in GitHub Secrets
3. Verify workflow runs in Actions tab
4. Test manually with `workflow_dispatch` button
5. Monitor first scheduled run
