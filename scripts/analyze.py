#!/usr/bin/env python3
"""Analyze scraped JSON data from stdin.

Usage: goscrape run --url <URL> --python scripts/analyze.py

Outputs a summary: page count, avg title length, domain stats, link distribution.
"""

import json
import sys
from collections import Counter
from urllib.parse import urlparse


def analyze(data):
    if not data:
        return {"error": "no data"}

    pages = data if isinstance(data, list) else [data]

    result = {
        "total_pages": len(pages),
        "domains": Counter(),
        "avg_title_length": 0,
        "total_links": 0,
        "total_files": 0,
        "pages_with_errors": 0,
        "status_codes": Counter(),
    }

    title_lengths = []
    for page in pages:
        parsed = urlparse(page.get("url", ""))
        result["domains"][parsed.netloc] += 1
        title_lengths.append(len(page.get("title", "")))
        result["total_links"] += len(page.get("links", []))
        result["total_files"] += len(page.get("files", []))
        result["status_codes"][page.get("status_code", 0)] += 1
        if page.get("error"):
            result["pages_with_errors"] += 1

    if title_lengths:
        result["avg_title_length"] = sum(title_lengths) / len(title_lengths)

    result["domains"] = dict(result["domains"].most_common())
    result["status_codes"] = dict(result["status_codes"].most_common())

    return result


if __name__ == "__main__":
    raw = sys.stdin.read()
    try:
        data = json.loads(raw)
    except json.JSONDecodeError as e:
        print(json.dumps({"error": f"invalid JSON: {e}"}))
        sys.exit(1)

    summary = analyze(data)
    print(json.dumps(summary, indent=2))
