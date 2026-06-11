#!/usr/bin/env python3
"""Export scraped data to Excel (.xlsx).

Usage: goscrape run --url <URL> --python scripts/export_excel.py

Outputs an Excel file with scraped pages and file references.
"""

import json
import sys
from datetime import datetime


def export_excel(data):
    try:
        import openpyxl
    except ImportError:
        print(
            json.dumps(
                {
                    "error": "openpyxl not installed. Run: pip install openpyxl"
                }
            )
        )
        sys.exit(1)

    pages = data if isinstance(data, list) else [data]

    wb = openpyxl.Workbook()
    ws = wb.active
    ws.title = "Pages"

    ws.append(["URL", "Title", "Status", "Links", "Files", "Error"])

    for page in pages:
        ws.append(
            [
                page.get("url", ""),
                page.get("title", ""),
                page.get("status_code", 0),
                len(page.get("links", [])),
                len(page.get("files", [])),
                page.get("error", ""),
            ]
        )

    files_ws = wb.create_sheet("Files")
    files_ws.append(["Page URL", "File URL", "Filename", "Type"])
    for page in pages:
        for f in page.get("files", []):
            files_ws.append(
                [
                    page.get("url", ""),
                    f.get("url", ""),
                    f.get("filename", ""),
                    f.get("type", ""),
                ]
            )

    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    filename = f"goscrape_export_{timestamp}.xlsx"
    wb.save(filename)

    return {"file": filename, "pages": len(pages), "files_sheet": len(pages)}


if __name__ == "__main__":
    raw = sys.stdin.read()
    try:
        data = json.loads(raw)
    except json.JSONDecodeError as e:
        print(json.dumps({"error": f"invalid JSON: {e}"}))
        sys.exit(1)

    result = export_excel(data)
    print(json.dumps(result, indent=2))
