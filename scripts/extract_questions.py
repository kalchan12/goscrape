#!/usr/bin/env python3
"""Extract exam questions from Next.js RSC payload.

Usage: curl -s <exam_url> | python3 scripts/extract_questions.py > questions.json

Or with goscrape:
  goscrape run --url <exam_url> --python scripts/extract_questions.py
"""

import json
import re
import sys


def extract_questions(html):
    # Find the questions JSON in the RSC payload.
    # The data is inside __next_f.push calls as doubly-escaped JSON.

    # First, try to find data chunks that look like question arrays
    matches = re.findall(
        r'"questionKey"[^}]+"correctAnswers"[^}]+"explanation"[^}]+"selectionMode"',
        html,
    )

    if matches:
        # Found inline question fragments - rebuild the array
        questions = []
        for m in matches:
            # Fix escaping
            cleaned = m.replace('\\"', '"').replace('\\n', '\n')
            # Try to extract individual fields
            try:
                qkey = re.search(r'"questionKey"\s*:\s*"([^"]+)"', cleaned)
                sid = re.search(r'"sourceId"\s*:\s*(\d+)', cleaned)
                qtext = re.search(
                    r'"question"\s*:\s*\[(.*?)\]\s*,\s*"options"', cleaned
                )
                opts = re.search(
                    r'"options"\s*:\s*\[(.*?)\]\s*,\s*"correctAnswers"', cleaned
                )
                cans = re.search(r'"correctAnswers"\s*:\s*\[([^\]]+)\]', cleaned)
                expl = re.search(r'"explanation"\s*:\s*"([^"]*)"', cleaned)
                sel = re.search(r'"selectionMode"\s*:\s*"([^"]+)"', cleaned)

                q = {
                    "questionKey": qkey.group(1) if qkey else "",
                    "sourceId": int(sid.group(1)) if sid else 0,
                    "question_text": extract_text_blocks(qtext.group(1))
                    if qtext
                    else "",
                    "options": extract_options(opts.group(1)) if opts else [],
                    "correctAnswers": (
                        [int(x.strip()) for x in cans.group(1).split(",")]
                        if cans
                        else []
                    ),
                    "explanation": expl.group(1) if expl else "",
                    "selectionMode": sel.group(1) if sel else "single",
                }
                questions.append(q)
            except Exception as e:
                pass
        return questions

    # Fallback: find all correctAnswers and rebuild context
    lines = html.split("\\n")
    full_text = "\n".join(lines)

    # Look for a large JSON array structure
    idx = full_text.find('"questionKey"')
    if idx == -1:
        return None

    # Try to extract the containing array
    start = full_text.rfind("[", 0, idx)
    if start == -1:
        start = idx

    # Find the end - look for "]}]" pattern that closes the array
    end = full_text.find("}]", idx)
    if end == -1:
        end = len(full_text)
    else:
        end += 2

    raw = full_text[start:end]
    cleaned = raw.replace('\\"', '"').replace('\\n', "\n")

    try:
        return json.loads(cleaned)
    except json.JSONDecodeError:
        # Partial parse - use regex to extract individual questions
        return questions_from_partial(cleaned)


def extract_text_blocks(raw):
    """Extract text from blocks array."""
    texts = re.findall(r'"text"\s*:\s*"([^"]*)"', raw)
    return " ".join(texts) if texts else ""


def extract_options(raw):
    """Extract options from options array."""
    options = []
    # Match key + blocks pattern
    parts = re.findall(
        r'\{\s*"key"\s*:\s*"([^"]*)"\s*,\s*"blocks"\s*:\s*\[(.*?)\]\s*\}', raw
    )
    for key, blocks in parts:
        text = extract_text_blocks(blocks)
        options.append({"key": key, "text": text})
    return options


def questions_from_partial(text):
    """Parse questions from partially cleaned text."""
    questions = []
    # Split by questionKey
    parts = text.split('"questionKey"')
    for part in parts[1:]:
        try:
            qkey = re.search(r'"\s*:\s*"([^"]+)"', part)
            sid = re.search(r'"sourceId"\s*:\s*(\d+)', part)

            qtext_m = re.search(
                r'"question"\s*:\s*\[\s*\{[^}]*"text"\s*:\s*"([^"]*)"', part
            )

            opts = re.findall(
                r'"key"\s*:\s*"([^"]*)"[^}]*"text"\s*:\s*"([^"]*)"', part
            )
            options = [{"key": k, "text": t} for k, t in opts]

            cans = re.search(r'"correctAnswers"\s*:\s*\[([^\]]+)\]', part)
            expl = re.search(r'"explanation"\s*:\s*"([^"]*)"', part)
            sel = re.search(r'"selectionMode"\s*:\s*"([^"]+)"', part)

            q = {
                "questionKey": qkey.group(1) if qkey else "",
                "sourceId": int(sid.group(1)) if sid else 0,
                "question_text": qtext_m.group(1) if qtext_m else "",
                "options": options,
                "correctAnswers": (
                    [int(x.strip()) for x in cans.group(1).split(",")] if cans else []
                ),
                "explanation": expl.group(1) if expl else "",
                "selectionMode": sel.group(1) if sel else "single",
            }
            questions.append(q)
        except Exception:
            pass
    return questions


if __name__ == "__main__":
    html = sys.stdin.read()
    questions = extract_questions(html)
    if questions:
        print(json.dumps(questions, indent=2, ensure_ascii=False))
        print(f"\n// Extracted {len(questions)} questions", file=sys.stderr)
    else:
        print(json.dumps({"error": "no questions found"}), file=sys.stderr)
        sys.exit(1)
