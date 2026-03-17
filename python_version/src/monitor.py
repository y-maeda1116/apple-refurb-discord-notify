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
APPLE_URL = "https://www.apple.com/jp/shop/refurbished/mac/mac-mini"
INVENTORY_FILE = Path("../inventory.json")
DISCORD_WEBHOOK_URL = os.environ.get("DISCORD_WEBHOOK_URL")

if not DISCORD_WEBHOOK_URL:
    logger.error("DISCORD_WEBHOOK_URL environment variable not set")
    raise SystemExit(1)


def fetch_products() -> list[dict]:
    """Fetch products from Apple's refurbished page by parsing HTML.

    Returns:
        List of product dictionaries from Apple API.

    Raises:
        requests.RequestException: If API request fails.
    """
    logger.info(f"Fetching products from {APPLE_URL}")

    try:
        response = requests.get(APPLE_URL, timeout=30)
        response.raise_for_status()
        html = response.text

        # Extract products array from embedded JSON
        pattern = r'products"\s*:\s*(\[[^\]]+(?:\[[^\]]*\][^\]]*)*\])'
        match = re.search(pattern, html)

        if not match:
            logger.warning("Could not find products data in HTML")
            return []

        products_json = match.group(1)
        
        # Fix JSON syntax (JavaScript to Python)
        products_json = re.sub(r'(\w+)\s*:', r'"\1":', products_json)
        products_json = re.sub(r"'", r'"', products_json)
        
        products = json.loads(products_json)
        logger.info(f"Fetched {len(products)} products from Apple")
        return products

    except requests.RequestException as e:
        logger.error(f"Failed to fetch products: {e}")
        raise
    except json.JSONDecodeError as e:
        logger.error(f"Failed to parse JSON response: {e}")
        raise


def extract_ram(product: dict) -> int | None:
    """Extract RAM value in GB from product data.

    Args:
        product: Product dictionary from Apple API.

    Returns:
        RAM value in GB, or None if not found.
    """
    # Try tsMemorySize (Apple's unified memory field)
    dimensions = product.get('dimensions', {})
    ram_str = dimensions.get('tsMemorySize', '')
    
    if ram_str:
        match = re.search(r'(\d+)', ram_str.lower())
        if match:
            return int(match.group(1))

    # Fallback to other fields
    for field in ['memorySize', 'ram']:
        if field in dimensions:
            ram_str = str(dimensions[field])
            match = re.search(r'(\d+)', ram_str.lower())
            if match:
                return int(match.group(1))

    # Try regex on model name
    model = product.get('dimensions', {}).get('refurbClearModel', '')
    patterns = [r'(\d+)gb', r'(\d+) gb', r'(\d+)ギガバイト']
    for pattern in patterns:
        match = re.search(pattern, model.lower())
        if match:
            return int(match.group(1))

    return None


def filter_products(products: list[dict]) -> list[dict]:
    """Filter products for Mac mini with 24GB+ RAM.

    Args:
        products: List of product dictionaries from Apple API.

    Returns:
        List of filtered products as normalized dictionaries.
    """
    filtered = []

    for product in products:
        # Check if product is Mac mini
        dimensions = product.get('dimensions', {})
        model = dimensions.get('refurbClearModel', '').lower()
        
        if 'mini' not in model and 'macmini' not in model:
            continue

        # Extract RAM
        ram_gb = extract_ram(product)
        if ram_gb is None or ram_gb < 24:
            continue

        # Build product URL
        product_id = product.get('productTile', {}).get('id', '')
        url = f"{APPLE_URL}?fproduct={product_id}" if product_id else APPLE_URL

        # Get price
        price_info = product.get('productTile', {}).get('price', {})
        current_price = price_info.get('currentPrice', '0')
        price_str = str(current_price).replace('¥', '').replace(',', '').replace('JPY', '').strip()
        
        try:
            price_raw = int(float(price_str)) if price_str else 0
            if price_raw == 0:
                continue
        except ValueError:
            continue

        # Get chip name from year and model
        year = dimensions.get('dimensionRelYear', '')
        chip = f"M{year[-2:]}" if year and len(year) >= 2 else "Unknown"

        normalized = {
            "name": f"Mac mini {chip}",
            "price": f"¥{price_raw:,}",
            "price_raw": price_raw,
            "url": url,
            "ram": f"{ram_gb}GB",
            "ram_gb": ram_gb,
            "chip": chip,
            "thumbnail": ""
        }

        filtered.append(normalized)
        logger.info(f"Matched: {normalized['name']} - {normalized['ram']} - {normalized['price']}")

    logger.info(f"Filtered to {len(filtered)} Mac mini products with 24GB+ RAM")
    return filtered


def generate_unique_id(product: dict) -> str:
    """Generate unique ID from product name, RAM, and price.

    Args:
        product: Normalized product dictionary.

    Returns:
        Unique ID string (16 characters).
    """
    key = f"{product['name']}_{product['ram_gb']}_{product['price_raw']}"
    return hashlib.sha256(key.encode()).hexdigest()[:16]


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
