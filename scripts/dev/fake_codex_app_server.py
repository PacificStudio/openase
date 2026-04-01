#!/usr/bin/env python3

import json
import sys
import time


def send(message: dict) -> None:
    sys.stdout.write(json.dumps(message) + "\n")
    sys.stdout.flush()


def build_turn_started(thread_id: str, turn_id: str) -> dict:
    return {
        "jsonrpc": "2.0",
        "method": "turn/started",
        "params": {
            "threadId": thread_id,
            "turn": {
                "id": turn_id,
                "status": "inProgress",
            },
        },
    }


def build_token_usage(thread_id: str, turn_id: str, turn_number: int) -> dict:
    input_tokens = 128 * turn_number
    output_tokens = 64 * turn_number
    total_tokens = input_tokens + output_tokens
    return {
        "jsonrpc": "2.0",
        "method": "thread/tokenUsage/updated",
        "params": {
            "threadId": thread_id,
            "turnId": turn_id,
            "tokenUsage": {
                "total": {
                    "inputTokens": input_tokens,
                    "outputTokens": output_tokens,
                    "totalTokens": total_tokens,
                },
                "last": {
                    "inputTokens": input_tokens,
                    "outputTokens": output_tokens,
                    "totalTokens": total_tokens,
                },
            },
        },
    }


def build_completed_item(thread_id: str, turn_id: str, turn_number: int) -> dict:
    return {
        "jsonrpc": "2.0",
        "method": "item/completed",
        "params": {
            "threadId": thread_id,
            "turnId": turn_id,
            "item": {
                "id": f"assistant-message-{turn_number}",
                "type": "agentMessage",
                "text": "Fake Codex executed the requested turn.",
                "phase": "final",
            },
        },
    }


def build_turn_completed(thread_id: str, turn_id: str) -> dict:
    return {
        "jsonrpc": "2.0",
        "method": "turn/completed",
        "params": {
            "threadId": thread_id,
            "turn": {
                "id": turn_id,
                "status": "completed",
            },
        },
    }


def main() -> int:
    thread_id = "fake-thread-1"
    turn_number = 0

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
                        "userAgent": "fake-codex/0.2",
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

        if method == "turn/start":
            turn_number += 1
            turn_id = f"fake-turn-{turn_number}"

            send(
                {
                    "jsonrpc": "2.0",
                    "id": message.get("id"),
                    "result": {
                        "turn": {
                            "id": turn_id,
                            "status": "inProgress",
                        }
                    },
                }
            )
            send(build_turn_started(thread_id, turn_id))
            send(build_token_usage(thread_id, turn_id, turn_number))
            send(build_completed_item(thread_id, turn_id, turn_number))
            time.sleep(0.2)
            send(build_turn_completed(thread_id, turn_id))
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
