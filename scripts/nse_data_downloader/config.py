"""
Configuration for the NSE data downloader.
Update these values with your ICICI Direct Breeze API credentials.
"""

# ── Breeze API Credentials ──────────────────────────────────────────────
# Get these from: https://api.icicidirect.com/apiuser/login
API_KEY = ""          # Your Breeze API key
API_SECRET = ""       # Your Breeze API secret
SESSION_TOKEN = ""    # Session token (generated daily via TOTP login)

# ── Download Settings ───────────────────────────────────────────────────
INTERVAL = "1minute"           # "1second", "1minute", "5minute", "30minute", "1day"
EXCHANGE_CODE = "NSE"          # "NSE", "BSE"
PRODUCT_TYPE = "cash"          # "cash", "futures", "options"

# How many years back to fetch (API may only have ~3 years for 1min data)
YEARS_BACK = 10

# ── Rate Limiting ───────────────────────────────────────────────────────
CALLS_PER_MINUTE = 90          # API limit is 100/min, keep margin
CALLS_PER_DAY = 4800           # API limit is 5000/day, keep margin
SLEEP_BETWEEN_CALLS = 0.7      # seconds between API calls (~85/min)

# ── Storage ─────────────────────────────────────────────────────────────
# Data is saved as Parquet files (much smaller than CSV for 10 years of 1min data)
# Directory structure: DATA_DIR/{stock_code}/YYYY.parquet
DATA_DIR = "data"
OUTPUT_FORMAT = "parquet"      # "parquet" or "csv"

# ── Retry Settings ──────────────────────────────────────────────────────
MAX_RETRIES = 3
RETRY_BACKOFF = 2              # seconds, doubles each retry

# ── NSE Sector Indices ──────────────────────────────────────────────────
# These are the ICICI Breeze stock_codes for NSE sector indices.
# Verify against the Security Master file - codes may change.
SECTOR_INDICES = [
    "NIFTY",           # Nifty 50
    "BANKNIFTY",       # Nifty Bank
    "CNXIT",           # Nifty IT
    "CNXAUTO",         # Nifty Auto
    "CNXENERGY",       # Nifty Energy
    "CNXFIN",          # Nifty Financial Services
    "CNXFMCG",         # Nifty FMCG
    "CNXMEDIA",        # Nifty Media
    "CNXMETAL",        # Nifty Metal
    "CNXPHARMA",       # Nifty Pharma
    "CNXPSUBANK",      # Nifty PSU Bank
    "CNXREALTY",       # Nifty Realty
    "CNXPVTBANK",      # Nifty Private Bank
    "CNXCOMMODITIES",  # Nifty Commodities
    "CNXCONSUMPTION",  # Nifty Consumption
    "CNXINFRA",        # Nifty Infrastructure
    "CNXMNC",          # Nifty MNC
    "CNXSERVICE",      # Nifty Services Sector
    "NIFTYMID50",      # Nifty Midcap 50
    "NIFTYSMALL100",   # Nifty Smallcap 100
    "NIFTYNEXT50",     # Nifty Next 50
]
