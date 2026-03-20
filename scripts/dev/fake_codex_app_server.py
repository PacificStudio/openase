#!/usr/bin/env python3

import json
import sys


def send(message: dict) -> None:
    sys.stdout.write(json.dumps(message) + "\n")
    sys.stdout.flush()


def main() -> int:
    thread_id = "fake-thread-1"

    for raw_line in sys.stdin:
        line = raw_line.strip()
        if not line:
            continue

        message = json.loads(line)
        method = message.get("method", "")

        if method == "initialize":
            send(
                {
                    "jsonrpc": "2.0",
                    "id": message.get("id"),
                    "result": {
                        "userAgent": "fake-codex/0.1",
                        "platformFamily": "unix",
                        "platformOs": "linux",
                    },
                }
            )
            continue

        if method == "initialized":
            continue

        if method == "thread/start":
            send(
                {
                    "jsonrpc": "2.0",
                    "id": message.get("id"),
                    "result": {
                        "thread": {
                            "id": thread_id,
                        }
                    },
                }
            )
            continue

        if "id" in message:
            send(
                {
                    "jsonrpc": "2.0",
                    "id": message.get("id"),
                    "error": {
                        "code": -32601,
                        "message": f"unsupported fake codex method {method}",
                    },
                }
            )

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
