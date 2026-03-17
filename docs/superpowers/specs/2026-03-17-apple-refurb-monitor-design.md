# Apple Refurbished Mac mini Monitor Design

**Date:** 2026-03-17
**Status:** Approved
**Approach:** Single Script + GitHub Actions

## Overview

Monitor Apple's refurbished products page and send Discord notifications when Mac mini models with 24GB+ RAM become available. The system runs every 15 minutes via GitHub Actions, tracks notified products to avoid duplicate notifications, and detects restocks.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     GitHub Actions (cron: */15 * * * *)     │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│  src/monitor.py                                            │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │  fetch   │→ │  filter  │→ │  compare │→ │  notify  │   │
│  │   Apple  │  │  24GB+   │  │ inventory│  │ Discord  │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
│       │                                    │                │
│       ▼                                    ▼                │
│  https://apple.com/...              DISCORD_WEBHOOK_URL     │
│       │                                    │                │
│       ▼                                    │                │
│  inventory.json ←──────────────────────────┘                │
│  (commit if changed)                                     │
└─────────────────────────────────────────────────────────────┘
```

## Components

### src/monitor.py

**Functions:**

| Function | Description |
|----------|-------------|
| `fetch_products()` | GET Apple JSON API and parse product list |
| `filter_products(products)` | Extract Mac mini with RAM 24GB+ |
| `load_inventory()` | Load inventory.json, return as set |
| `save_inventory(inventory)` | Save inventory.json |
| `should_notify(item, inventory)` | Determine if notification needed (new or restocked) |
| `send_discord_embed(item, webhook_url, status)` | Send Discord embed notification |
| `commit_inventory()` | Git commit inventory.json if changed |

**Data Structures:**

```python
# Product from Apple API
{
    "name": "Mac mini M4",
    "price": "¥120,000",
    "url": "https://apple.com/jp/...",
    "ram": "24GB"
}

# inventory.json format
{
  "products": {
    "mac_mini_m4_24gb": {
      "name": "Mac mini M4",
      "ram": "24GB",
      "first_notified": "2026-03-17T10:00:00Z",
      "last_notified": "2026-03-17T10:00:00Z",
      "in_stock": true
    }
  },
  "last_fetch": "2026-03-17T10:00:00Z"
}
```

## Data Flow

```
1. fetch_products()
   └── GET https://www.apple.com/jp/shop/refurbished/ajax/mac/mac-mini
      └→ Parse response JSON

2. filter_products(products)
   └→ For each product:
      ├─ Contains "Mac mini" in name?
      ├─ Extract RAM from specs
      └─ RAM >= 24GB?
      └→ Return filtered product list

3. load_inventory()
   └→ Load notified product unique_ids from inventory.json

4. For each product:
   ├─ unique_id = hash(name + ram + ...)
   ├─ if unique_id not in inventory:
   │   └→ New notification
   ├─ else if not in_stock:
   │   └→ Restock notification
   └→ send_discord_embed()

5. save_inventory(inventory)
   └── Append notified products to inventory.json

6. commit_inventory()
   └── git diff --quiet inventory.json
      ├─ If changes:
      │   ├─ git add inventory.json
      │   ├─ git commit -m "chore: update inventory [skip ci]"
      │   └─ git push
      └─ If no changes: nothing
```

## Restock Detection Logic

```python
# Mark products that disappeared as out of stock
for unique_id in inventory["products"]:
    if unique_id not in current_product_ids:
        inventory["products"][unique_id]["in_stock"] = False

# Detect restocks
for product in filtered_products:
    unique_id = generate_unique_id(product)

    if unique_id in inventory["products"]:
        if not inventory["products"][unique_id]["in_stock"]:
            status = "reinstated"  # Was out of stock, now back
        else:
            continue  # Still in stock, no notification
    else:
        status = "new"  # Never seen before

    send_discord_embed(product, status)
```

## Error Handling

| Error Type | Handling |
|------------|----------|
| Apple API access failure (HTTP error) | Catch with try-except, log to GitHub Actions, exit with non-zero |
| JSON parse failure | Catch `json.JSONDecodeError`, log and exit |
| Discord webhook failure | Catch `requests.exceptions.RequestException`, log but save inventory |
| inventory.json missing | Create empty dict as initial state |
| Git operation failure | Catch `subprocess.CalledProcessError`, log |

**Error Notification Policy:** Per requirements, errors are NOT sent to Discord. GitHub Actions logs only.

## Discord Embed Design

```json
{
  "embeds": [{
    "title": "🍎 整備済製品: Mac mini M4 Pro 24GB",
    "url": "https://www.apple.com/jp/shop/refurbished/...",
    "color": 5814783,
    "thumbnail": {
      "url": "https://store.storeimages.cdn-apple.com/..."
    },
    "fields": [
      {
        "name": "💰 価格",
        "value": "¥138,800",
        "inline": true
      },
      {
        "name": "💾 RAM",
        "value": "24GB",
        "inline": true
      },
      {
        "name": "🔁 状態",
        "value": "再入庫 🆕",
        "inline": true
      }
    ],
    "footer": {
      "text": "Apple Refurbished Monitor | 2026-03-17 10:00"
    }
  }]
}
```

- Color: Apple Green (#589632)
- Status field: "新着" for new, "再入庫 🆕" for restocks
- Footer includes notification timestamp

## GitHub Actions Configuration

```yaml
name: Monitor Apple Refurbished Mac mini

on:
  schedule:
    - cron: '*/15 * * * *'
  workflow_dispatch:

jobs:
  monitor:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          token: ${{ secrets.GITHUB_TOKEN }}

      - name: Set up Python
        uses: actions/setup-python@v5
        with:
          python-version: '3.11'

      - name: Install dependencies
        run: pip install -r requirements.txt

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

## Directory Structure

```
apple-refurb-discord-notify/
├── .github/
│   └── workflows/
│       └── monitor.yml
├── src/
│   └── monitor.py
├── inventory.json
├── .env.example
├── .gitignore
├── requirements.txt
└── README.md
```

## Requirements.txt

```
requests==2.31.0
```

## Filtering Criteria

1. Product name contains "Mac mini"
2. RAM (Unified Memory) is 24GB or higher
3. Applies to all Mac mini models (M4, M4 Pro, etc.) as long as RAM >= 24GB

## Testing

**Local Verification:**
```bash
python -m venv venv
source venv/bin/activate
pip install -r requirements.txt
python src/monitor.py  # Single run test

rm inventory.json  # Test new notification
python src/monitor.py

python src/monitor.py  # Test duplicate prevention
```

**GitHub Actions Verification:**
1. Set `DISCORD_WEBHOOK_URL` in GitHub Secrets
2. Push `.github/workflows/monitor.yml`
3. Verify execution in Actions tab

## Requirements Summary

- **Language:** Python 3.11+
- **Library:** requests
- **Target URL:** https://www.apple.com/jp/shop/refurbished/ajax/mac/mac-mini
- **Filter:** Mac mini with RAM >= 24GB
- **Notification:** Discord Webhook with Embed format
- **State Management:** inventory.json with git commit on changes only
- **Schedule:** Every 15 minutes via GitHub Actions cron
- **Secret:** DISCORD_WEBHOOK_URL from GitHub Secrets
- **Price format:** 3-digit grouping (¥100,000)
- **Restock detection:** Enabled
- **Error notifications to Discord:** Disabled
