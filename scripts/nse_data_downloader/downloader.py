#!/usr/bin/env python3
"""
NSE Stock Data Downloader via ICICI Direct Breeze API

Downloads 1-minute OHLCV candle data for all NSE stocks and sector indices.
Handles rate limiting, date chunking (1000 candle limit per request),
resumable downloads, and saves as Parquet/CSV.

Usage:
    # Download all NSE stocks
    python downloader.py --all-stocks

    # Download sector indices only
    python downloader.py --indices

    # Download specific stocks
    python downloader.py --stocks RELIND TATSTE INFY

    # Download from Security Master file
    python downloader.py --from-master nse_master.csv

    # Resume interrupted download
    python downloader.py --all-stocks --resume

    # Dry run (show what would be downloaded)
    python downloader.py --all-stocks --dry-run
"""

import argparse
import csv
import json
import logging
import os
import sys
import time
from datetime import datetime, timedelta
from pathlib import Path

import pandas as pd

try:
    from breeze_connect import BreezeConnect
except ImportError:
    print("ERROR: breeze-connect not installed. Run: pip install breeze-connect")
    sys.exit(1)

import config

# ── Logging ─────────────────────────────────────────────────────────────
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    handlers=[
        logging.StreamHandler(),
        logging.FileHandler("downloader.log"),
    ],
)
log = logging.getLogger(__name__)

# ── Constants ───────────────────────────────────────────────────────────
# NSE trading hours: 9:15 AM - 3:30 PM = 375 minutes
TRADING_MINUTES_PER_DAY = 375
MAX_CANDLES_PER_REQUEST = 1000
# For 1-minute interval: ~2.6 trading days per request
TRADING_DAYS_PER_CHUNK = 2  # conservative: 2 days per request = 750 candles

# NSE holidays (2024-2026) - update as needed
# Source: https://www.nseindia.com/resources/exchange-communication-holidays
NSE_HOLIDAYS = {
    # 2024
    "2024-01-26", "2024-03-08", "2024-03-25", "2024-03-29",
    "2024-04-11", "2024-04-14", "2024-04-17", "2024-04-21",
    "2024-05-01", "2024-05-23", "2024-06-17", "2024-07-17",
    "2024-08-15", "2024-09-16", "2024-10-02", "2024-10-12",
    "2024-11-01", "2024-11-15", "2024-12-25",
    # 2025
    "2025-02-26", "2025-03-14", "2025-03-31", "2025-04-10",
    "2025-04-14", "2025-04-18", "2025-05-01", "2025-05-12",
    "2025-06-26", "2025-07-16", "2025-08-15", "2025-08-27",
    "2025-10-02", "2025-10-21", "2025-10-22", "2025-11-05",
    "2025-11-26", "2025-12-25",
    # 2026 (partial - update when available)
    "2026-01-26", "2026-03-10", "2026-03-19", "2026-04-02",
    "2026-04-03", "2026-04-14", "2026-05-01", "2026-08-15",
    "2026-10-02", "2026-12-25",
}


def is_trading_day(dt: datetime) -> bool:
    """Check if a date is an NSE trading day (not weekend, not holiday)."""
    if dt.weekday() >= 5:  # Saturday=5, Sunday=6
        return False
    return dt.strftime("%Y-%m-%d") not in NSE_HOLIDAYS


class RateLimiter:
    """Enforces Breeze API rate limits: calls/minute and calls/day."""

    def __init__(self, per_minute: int, per_day: int, sleep_between: float):
        self.per_minute = per_minute
        self.per_day = per_day
        self.sleep_between = sleep_between
        self.minute_calls = []
        self.day_count = 0
        self.day_start = datetime.now()

    def wait(self):
        """Block until we can safely make another API call."""
        now = datetime.now()

        # Reset daily counter at midnight
        if (now - self.day_start).total_seconds() > 86400:
            self.day_count = 0
            self.day_start = now

        # Check daily limit
        if self.day_count >= self.per_day:
            wait_secs = 86400 - (now - self.day_start).total_seconds()
            log.warning(f"Daily limit reached ({self.per_day}). "
                        f"Waiting {wait_secs/3600:.1f} hours until reset.")
            print(f"\n*** DAILY API LIMIT REACHED. Resume tomorrow or wait {wait_secs/3600:.1f}h ***")
            time.sleep(max(wait_secs, 0) + 60)
            self.day_count = 0
            self.day_start = datetime.now()

        # Enforce per-minute limit
        cutoff = now - timedelta(seconds=60)
        self.minute_calls = [t for t in self.minute_calls if t > cutoff]
        if len(self.minute_calls) >= self.per_minute:
            oldest = self.minute_calls[0]
            wait_secs = 60 - (now - oldest).total_seconds() + 0.5
            if wait_secs > 0:
                log.debug(f"Rate limit: waiting {wait_secs:.1f}s")
                time.sleep(wait_secs)

        # Minimum sleep between calls
        time.sleep(self.sleep_between)

        self.minute_calls.append(datetime.now())
        self.day_count += 1

    @property
    def calls_remaining_today(self) -> int:
        return self.per_day - self.day_count


class BreezeDownloader:
    """Downloads historical OHLCV data from ICICI Direct Breeze API."""

    def __init__(self, dry_run: bool = False):
        self.dry_run = dry_run
        self.data_dir = Path(config.DATA_DIR)
        self.data_dir.mkdir(parents=True, exist_ok=True)
        self.progress_file = self.data_dir / "_progress.json"
        self.progress = self._load_progress()
        self.rate_limiter = RateLimiter(
            config.CALLS_PER_MINUTE,
            config.CALLS_PER_DAY,
            config.SLEEP_BETWEEN_CALLS,
        )
        self.breeze = None
        self.stats = {"api_calls": 0, "candles_downloaded": 0, "stocks_completed": 0, "errors": 0}

    def connect(self):
        """Initialize and authenticate with Breeze API."""
        if self.dry_run:
            log.info("DRY RUN: Skipping API connection")
            return

        if not config.API_KEY or not config.API_SECRET or not config.SESSION_TOKEN:
            print("\n" + "=" * 60)
            print("ERROR: API credentials not configured!")
            print("=" * 60)
            print("\nSteps to set up:")
            print("1. Register at https://api.icicidirect.com/apiuser/login")
            print("2. Get your API Key and Secret from the dashboard")
            print("3. Generate a session token (valid for 1 day)")
            print("4. Update config.py with your credentials")
            print("=" * 60 + "\n")
            sys.exit(1)

        log.info("Connecting to Breeze API...")
        self.breeze = BreezeConnect(api_key=config.API_KEY)
        self.breeze.generate_session(
            api_secret=config.API_SECRET,
            session_token=config.SESSION_TOKEN,
        )
        log.info("Connected successfully")

    def _load_progress(self) -> dict:
        """Load download progress for resume support."""
        if self.progress_file.exists():
            with open(self.progress_file) as f:
                return json.load(f)
        return {}

    def _save_progress(self):
        """Save download progress."""
        with open(self.progress_file, "w") as f:
            json.dump(self.progress, f, indent=2, default=str)

    def _get_date_chunks(self, start_date: datetime, end_date: datetime) -> list[tuple[datetime, datetime]]:
        """
        Split date range into chunks that yield < 1000 candles each.
        For 1-minute data: each chunk = 2 trading days.
        """
        chunks = []
        current = start_date

        while current < end_date:
            # Find the end of this chunk (TRADING_DAYS_PER_CHUNK trading days ahead)
            chunk_end = current
            trading_days_found = 0

            while trading_days_found < TRADING_DAYS_PER_CHUNK and chunk_end < end_date:
                chunk_end += timedelta(days=1)
                if is_trading_day(chunk_end):
                    trading_days_found += 1

            # Don't exceed the overall end date
            chunk_end = min(chunk_end, end_date)
            chunks.append((current, chunk_end))
            current = chunk_end + timedelta(days=1)

        return chunks

    def _fetch_historical(self, stock_code: str, from_date: str, to_date: str,
                          product_type: str = None) -> pd.DataFrame | None:
        """
        Make a single API call for historical data.
        Returns DataFrame or None on failure.
        """
        if self.dry_run:
            return pd.DataFrame()

        self.rate_limiter.wait()

        for attempt in range(config.MAX_RETRIES):
            try:
                resp = self.breeze.get_historical_data_v2(
                    interval=config.INTERVAL,
                    from_date=from_date,
                    to_date=to_date,
                    stock_code=stock_code,
                    exchange_code=config.EXCHANGE_CODE,
                    product_type=product_type or config.PRODUCT_TYPE,
                )
                self.stats["api_calls"] += 1

                if resp and resp.get("Success"):
                    data = resp["Success"]
                    if data:
                        df = pd.DataFrame(data)
                        self.stats["candles_downloaded"] += len(df)
                        return df
                    return pd.DataFrame()

                if resp and resp.get("Error"):
                    error_msg = resp["Error"]
                    log.warning(f"API error for {stock_code} ({from_date} to {to_date}): {error_msg}")
                    if "session" in str(error_msg).lower() or "token" in str(error_msg).lower():
                        log.error("Session expired! Generate a new session token and update config.py")
                        sys.exit(1)
                    return pd.DataFrame()

                return pd.DataFrame()

            except Exception as e:
                log.warning(f"Attempt {attempt + 1}/{config.MAX_RETRIES} failed for "
                            f"{stock_code}: {e}")
                if attempt < config.MAX_RETRIES - 1:
                    backoff = config.RETRY_BACKOFF * (2 ** attempt)
                    time.sleep(backoff)
                else:
                    log.error(f"All retries failed for {stock_code} ({from_date} to {to_date})")
                    self.stats["errors"] += 1
                    return None

    def download_stock(self, stock_code: str, resume: bool = False,
                       product_type: str = None) -> bool:
        """
        Download all historical 1-min data for a single stock.
        Saves incrementally per-year as Parquet/CSV.
        """
        end_date = datetime.now()
        start_date = end_date - timedelta(days=365 * config.YEARS_BACK)

        # Check resume state
        last_downloaded = None
        if resume and stock_code in self.progress:
            last_date_str = self.progress[stock_code].get("last_date")
            if last_date_str:
                last_downloaded = datetime.fromisoformat(last_date_str)
                start_date = last_downloaded + timedelta(days=1)
                log.info(f"Resuming {stock_code} from {start_date.date()}")

        if start_date >= end_date:
            log.info(f"{stock_code}: Already up to date")
            return True

        stock_dir = self.data_dir / stock_code
        stock_dir.mkdir(parents=True, exist_ok=True)

        chunks = self._get_date_chunks(start_date, end_date)
        total_chunks = len(chunks)

        log.info(f"Downloading {stock_code}: {start_date.date()} to {end_date.date()} "
                 f"({total_chunks} API calls needed)")

        if self.dry_run:
            return True

        all_data = []
        current_year = None

        for i, (chunk_start, chunk_end) in enumerate(chunks):
            # Format dates as required by Breeze API: "YYYY-MM-DDT07:00:00.000Z"
            from_str = chunk_start.strftime("%Y-%m-%dT07:00:00.000Z")
            to_str = chunk_end.strftime("%Y-%m-%dT16:00:00.000Z")

            df = self._fetch_historical(stock_code, from_str, to_str, product_type)

            if df is not None and not df.empty:
                all_data.append(df)

            # Save data when year changes or at the end
            chunk_year = chunk_end.year
            if current_year is not None and chunk_year != current_year:
                self._save_year_data(stock_code, current_year, all_data, stock_dir)
                all_data = []

            current_year = chunk_year

            # Update progress
            self.progress[stock_code] = {
                "last_date": chunk_end.isoformat(),
                "status": "in_progress",
            }
            if (i + 1) % 50 == 0:
                self._save_progress()
                log.info(f"  {stock_code}: {i + 1}/{total_chunks} chunks "
                         f"({(i + 1) / total_chunks * 100:.1f}%) | "
                         f"API calls today: {self.stats['api_calls']} | "
                         f"Remaining: {self.rate_limiter.calls_remaining_today}")

            # Progress indicator
            if (i + 1) % 10 == 0:
                print(f"\r  {stock_code}: {i + 1}/{total_chunks} "
                      f"({(i + 1) / total_chunks * 100:.0f}%)", end="", flush=True)

        # Save remaining data
        if all_data and current_year:
            self._save_year_data(stock_code, current_year, all_data, stock_dir)

        self.progress[stock_code] = {
            "last_date": end_date.isoformat(),
            "status": "completed",
        }
        self._save_progress()
        self.stats["stocks_completed"] += 1
        print(f"\r  {stock_code}: DONE ({total_chunks} chunks)")
        return True

    def _save_year_data(self, stock_code: str, year: int,
                        data_frames: list[pd.DataFrame], stock_dir: Path):
        """Concatenate and save data for a specific year."""
        if not data_frames:
            return

        frames_with_data = [df for df in data_frames if not df.empty]
        if not frames_with_data:
            return

        df = pd.concat(frames_with_data, ignore_index=True)

        # Filter to the target year
        if "datetime" in df.columns:
            df["datetime"] = pd.to_datetime(df["datetime"])
            df = df[df["datetime"].dt.year == year]
        elif "date" in df.columns:
            df["date"] = pd.to_datetime(df["date"])
            df = df[df["date"].dt.year == year]

        if df.empty:
            return

        # Remove duplicates
        df = df.drop_duplicates()

        # Sort by datetime
        time_col = "datetime" if "datetime" in df.columns else "date"
        df = df.sort_values(time_col).reset_index(drop=True)

        if config.OUTPUT_FORMAT == "parquet":
            filepath = stock_dir / f"{year}.parquet"
            # Merge with existing data if file exists
            if filepath.exists():
                existing = pd.read_parquet(filepath)
                df = pd.concat([existing, df], ignore_index=True)
                df = df.drop_duplicates().sort_values(time_col).reset_index(drop=True)
            df.to_parquet(filepath, index=False, compression="snappy")
        else:
            filepath = stock_dir / f"{year}.csv"
            if filepath.exists():
                existing = pd.read_csv(filepath)
                df = pd.concat([existing, df], ignore_index=True)
                df = df.drop_duplicates().sort_values(time_col).reset_index(drop=True)
            df.to_csv(filepath, index=False)

        log.debug(f"Saved {len(df)} rows to {filepath}")

    def get_all_nse_stocks(self) -> list[str]:
        """
        Get list of all NSE stock codes from the Security Master file.
        Downloads the file from Breeze API if not present locally.
        """
        master_file = self.data_dir / "nse_master.csv"

        if not master_file.exists():
            log.info("Downloading NSE Security Master file...")
            if self.dry_run:
                log.info("DRY RUN: Would download Security Master from "
                         "https://api.icicidirect.com/breezeapi/documents/index.html#instruments")
                return ["RELIND", "TATSTE", "INFY", "TCS", "HDFCBANK"]

            try:
                import requests
                # The Security Master URL - this may need updating
                url = "https://directlink.icicidirect.com/NewSiteLinks/NSEScripMaster.txt"
                resp = requests.get(url, timeout=30)
                resp.raise_for_status()
                master_file.write_text(resp.text)
                log.info(f"Security Master saved to {master_file}")
            except Exception as e:
                log.error(f"Failed to download Security Master: {e}")
                print("\nCould not auto-download the Security Master file.")
                print("Please download it manually from:")
                print("  https://api.icicidirect.com/breezeapi/documents/index.html#instruments")
                print(f"Save it as: {master_file}")
                sys.exit(1)

        return self._parse_master_file(master_file)

    def _parse_master_file(self, filepath: Path) -> list[str]:
        """Parse the Security Master file to extract stock codes."""
        stock_codes = set()

        try:
            # Try different encodings and delimiters
            for encoding in ["utf-8", "latin-1", "cp1252"]:
                try:
                    with open(filepath, encoding=encoding) as f:
                        reader = csv.reader(f)
                        header = next(reader, None)

                        if header is None:
                            continue

                        # Find the stock code column (usually "Short Name" or column index 1)
                        code_col = None
                        series_col = None
                        for i, h in enumerate(header):
                            h_lower = h.strip().lower()
                            if h_lower in ("short name", "shortname", "stock_code", "stockcode"):
                                code_col = i
                            if h_lower in ("series", "scrip_type"):
                                series_col = i

                        if code_col is None:
                            # Fallback: assume column 1 is the stock code
                            code_col = 1 if len(header) > 1 else 0

                        for row in reader:
                            if len(row) <= code_col:
                                continue
                            code = row[code_col].strip()
                            # Filter for EQ series only (equity)
                            if series_col is not None and len(row) > series_col:
                                if row[series_col].strip() not in ("EQ", "BE", ""):
                                    continue
                            if code and len(code) <= 20:
                                stock_codes.add(code)

                        break  # success
                except UnicodeDecodeError:
                    continue

        except Exception as e:
            log.error(f"Error parsing master file: {e}")
            sys.exit(1)

        codes = sorted(stock_codes)
        log.info(f"Found {len(codes)} stock codes from Security Master")
        return codes

    def download_all_stocks(self, resume: bool = False):
        """Download data for all NSE stocks."""
        stocks = self.get_all_nse_stocks()
        self._download_list(stocks, resume, label="NSE stocks")

    def download_indices(self, resume: bool = False):
        """Download data for all sector indices."""
        self._download_list(config.SECTOR_INDICES, resume,
                            label="sector indices", product_type="cash")

    def download_specific(self, stock_codes: list[str], resume: bool = False):
        """Download data for specific stocks."""
        self._download_list(stock_codes, resume, label="specified stocks")

    def _download_list(self, codes: list[str], resume: bool,
                       label: str, product_type: str = None):
        """Download data for a list of stock codes."""
        # Skip already completed stocks if resuming
        if resume:
            remaining = [c for c in codes
                         if self.progress.get(c, {}).get("status") != "completed"]
            log.info(f"Resuming: {len(codes) - len(remaining)} already completed, "
                     f"{len(remaining)} remaining")
            codes = remaining

        total = len(codes)
        log.info(f"Starting download of {total} {label}")

        # Estimate time
        if config.INTERVAL == "1minute":
            # ~500 chunks per stock for 10 years, ~0.7s per call
            est_calls_per_stock = (config.YEARS_BACK * 250) // TRADING_DAYS_PER_CHUNK
            est_time_per_stock = est_calls_per_stock * config.SLEEP_BETWEEN_CALLS
            est_total_hours = (est_time_per_stock * total) / 3600
            est_days = (est_calls_per_stock * total) / config.CALLS_PER_DAY
            print(f"\nEstimated: ~{est_calls_per_stock} API calls/stock, "
                  f"~{est_total_hours:.1f} hours total")
            print(f"At {config.CALLS_PER_DAY} calls/day limit: ~{est_days:.1f} days")
            print(f"(1-min data may only be available for ~3 years, reducing actual time)\n")

        for i, code in enumerate(codes):
            print(f"\n[{i + 1}/{total}] {code}")
            try:
                self.download_stock(code, resume=resume, product_type=product_type)
            except KeyboardInterrupt:
                log.info("Download interrupted by user. Progress saved.")
                self._save_progress()
                self._print_stats()
                sys.exit(0)
            except Exception as e:
                log.error(f"Failed to download {code}: {e}")
                self.stats["errors"] += 1
                continue

        self._save_progress()
        self._print_stats()

    def _print_stats(self):
        """Print download statistics."""
        print("\n" + "=" * 50)
        print("Download Statistics")
        print("=" * 50)
        print(f"  API calls made:      {self.stats['api_calls']}")
        print(f"  Candles downloaded:   {self.stats['candles_downloaded']:,}")
        print(f"  Stocks completed:     {self.stats['stocks_completed']}")
        print(f"  Errors:               {self.stats['errors']}")
        print(f"  Data saved to:        {self.data_dir.absolute()}")
        print("=" * 50)

    def estimate(self, stock_count: int):
        """Print download time and API call estimates."""
        if config.INTERVAL == "1minute":
            days_per_chunk = TRADING_DAYS_PER_CHUNK
        elif config.INTERVAL == "5minute":
            days_per_chunk = 3
        elif config.INTERVAL == "30minute":
            days_per_chunk = 20
        elif config.INTERVAL == "1day":
            days_per_chunk = 1000
        else:
            days_per_chunk = 2

        trading_days = config.YEARS_BACK * 250
        chunks_per_stock = trading_days // days_per_chunk
        total_calls = chunks_per_stock * stock_count

        print(f"\nDownload Estimate ({config.INTERVAL} interval, {config.YEARS_BACK} years)")
        print("=" * 50)
        print(f"  Stocks to download:   {stock_count}")
        print(f"  Trading days:         ~{trading_days}")
        print(f"  API calls/stock:      ~{chunks_per_stock}")
        print(f"  Total API calls:      ~{total_calls:,}")
        print(f"  At {config.CALLS_PER_MINUTE}/min:         "
              f"~{total_calls / config.CALLS_PER_MINUTE / 60:.1f} hours")
        print(f"  At {config.CALLS_PER_DAY}/day limit:      "
              f"~{total_calls / config.CALLS_PER_DAY:.1f} days")

        # Storage estimate
        # 1-min data: ~375 rows/day, ~50 bytes/row in Parquet
        rows_per_stock = trading_days * TRADING_MINUTES_PER_DAY
        if config.OUTPUT_FORMAT == "parquet":
            bytes_per_row = 50
        else:
            bytes_per_row = 120  # CSV
        total_gb = (rows_per_stock * stock_count * bytes_per_row) / (1024 ** 3)
        print(f"  Est. storage:         ~{total_gb:.1f} GB ({config.OUTPUT_FORMAT})")
        print("=" * 50)


def main():
    parser = argparse.ArgumentParser(
        description="Download NSE historical candle data from ICICI Direct Breeze API",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s --all-stocks              Download all NSE stocks
  %(prog)s --indices                 Download sector indices only
  %(prog)s --stocks RELIND TATSTE    Download specific stocks
  %(prog)s --all-stocks --resume     Resume interrupted download
  %(prog)s --estimate 1800           Estimate time for 1800 stocks
  %(prog)s --all-stocks --dry-run    Show plan without downloading
        """,
    )

    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("--all-stocks", action="store_true",
                       help="Download all NSE stocks (from Security Master)")
    group.add_argument("--indices", action="store_true",
                       help="Download NSE sector indices")
    group.add_argument("--stocks", nargs="+",
                       help="Download specific stock codes (ICICI format)")
    group.add_argument("--from-master", type=str,
                       help="Path to Security Master CSV file")
    group.add_argument("--estimate", type=int, metavar="N",
                       help="Estimate download time for N stocks")

    parser.add_argument("--resume", action="store_true",
                        help="Resume from last checkpoint")
    parser.add_argument("--dry-run", action="store_true",
                        help="Show what would be downloaded without calling API")

    args = parser.parse_args()

    downloader = BreezeDownloader(dry_run=args.dry_run)

    if args.estimate:
        downloader.estimate(args.estimate)
        return

    downloader.connect()

    if args.all_stocks:
        downloader.download_all_stocks(resume=args.resume)
    elif args.indices:
        downloader.download_indices(resume=args.resume)
    elif args.stocks:
        downloader.download_specific(args.stocks, resume=args.resume)
    elif args.from_master:
        codes = downloader._parse_master_file(Path(args.from_master))
        downloader._download_list(codes, args.resume, label="stocks from master file")


if __name__ == "__main__":
    main()
