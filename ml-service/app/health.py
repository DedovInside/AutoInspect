from __future__ import annotations

import json
import threading
import time
from dataclasses import dataclass
from http.server import BaseHTTPRequestHandler, HTTPServer
from typing import Callable


@dataclass
class HealthState:
    started_at: float = time.time()
    last_success_at: float | None = None
    last_error_at: float | None = None
    last_error_message: str | None = None

    def mark_success(self) -> None:
        self.last_success_at = time.time()

    def mark_error(self, message: str) -> None:
        self.last_error_at = time.time()
        self.last_error_message = message

    def snapshot(self) -> dict[str, object]:
        status = "ok"
        if self.last_error_at is not None and (
            self.last_success_at is None or self.last_error_at > self.last_success_at
        ):
            status = "degraded"
        return {
            "status": status,
            "started_at": self.started_at,
            "last_success_at": self.last_success_at,
            "last_error_at": self.last_error_at,
            "last_error_message": self.last_error_message,
        }


class _HealthHandler(BaseHTTPRequestHandler):
    def do_GET(self) -> None:  # noqa: N802
        if self.path not in ("/health", "/healthz"):
            self.send_response(404)
            self.end_headers()
            return

        payload = self.server.health_provider()
        body = json.dumps(payload).encode("utf-8")
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, format: str, *args: object) -> None:  # noqa: A002
        return


class HealthServer:
    def __init__(self, host: str, port: int, provider: Callable[[], dict[str, object]]):
        self._server = HTTPServer((host, port), _HealthHandler)
        self._server.health_provider = provider  # type: ignore[attr-defined]
        self._thread = threading.Thread(target=self._server.serve_forever, daemon=True)

    def start(self) -> None:
        self._thread.start()

    def stop(self) -> None:
        self._server.shutdown()
