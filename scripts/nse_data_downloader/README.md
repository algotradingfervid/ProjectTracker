# NSE Stock Data Downloader (ICICI Breeze API)

Downloads 1-minute OHLCV candle data for all NSE stocks and sector indices using the ICICI Direct Breeze API.

## Setup

```bash
cd scripts/nse_data_downloader
pip install -r requirements.txt
```

### API Credentials

1. **ICICI Direct account** with Breeze API enabled
2. Register at https://api.icicidirect.com/apiuser/login
3. Get your **API Key** and **API Secret** from the dashboard
4. Generate a **Session Token** (valid for 1 day — requires daily TOTP login)
5. Update `config.py` with your credentials

## Usage

```bash
# Estimate time/storage for all ~1800 NSE stocks
python downloader.py --estimate 1800

# Download all NSE stocks (fetches Security Master automatically)
python downloader.py --all-stocks

# Download sector indices only
python downloader.py --indices

# Download specific stocks (use ICICI stock codes, not NSE symbols)
python downloader.py --stocks RELIND TATSTE INFY TCS

# Resume an interrupted download
python downloader.py --all-stocks --resume

# Dry run (see what would happen without calling the API)
python downloader.py --all-stocks --dry-run
```

## API Limits & Estimates

| Parameter | Value |
|---|---|
| API calls/minute | 100 (we use 90 for safety) |
| API calls/day | 5,000 (we use 4,800) |
| Max candles/request | 1,000 |
| 1-min candles/trading day | 375 (9:15 AM - 3:30 PM) |
| Requests/stock (10 years) | ~1,250 |
| **Time per stock** | **~15 minutes** |
| **All NSE stocks (~1800)** | **~19 days** (at daily limit) |
| **Storage (Parquet)** | **~50 GB** for all stocks |

### Reality Check

- **1-minute data availability**: The API advertises 10 years, but practical availability for 1-min data appears to be **~3 years** for most stocks. Daily data goes back further.
- **Data is NOT adjusted** for corporate actions (splits, bonuses). You'll need to adjust manually.
- **Session token expires daily** — you must regenerate it each day and update `config.py`.

## Stock Codes

ICICI Breeze uses its own stock code format, NOT standard NSE ticker symbols:

| NSE Symbol | Breeze Code |
|---|---|
| RELIANCE | RELIND |
| TATASTEEL | TATSTE |
| HDFCBANK | HDFBAN |
| INFY | INFY |
| TCS | TCS |

The full mapping is in the **Security Master file**, downloaded automatically or from:
https://api.icicidirect.com/breezeapi/documents/index.html#instruments

## Data Output

```
data/
├── _progress.json          # Resume checkpoint
├── nse_master.csv          # Security Master (auto-downloaded)
├── RELIND/
│   ├── 2023.parquet
│   ├── 2024.parquet
│   └── 2025.parquet
├── TATSTE/
│   ├── 2023.parquet
│   └── ...
└── NIFTY/
    └── ...
```

Each Parquet/CSV file contains: `datetime, open, high, low, close, volume`

## Resumable Downloads

Progress is saved to `data/_progress.json`. If interrupted (Ctrl+C, daily limit, crash), just re-run with `--resume` to continue from where you left off.

## Configuration

Edit `config.py` to change:
- `INTERVAL` — candle interval (default: `"1minute"`)
- `YEARS_BACK` — how many years of data (default: 10)
- `OUTPUT_FORMAT` — `"parquet"` (recommended) or `"csv"`
- `DATA_DIR` — output directory (default: `"data"`)
- Rate limiting parameters
