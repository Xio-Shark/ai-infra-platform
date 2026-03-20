from __future__ import annotations

import logging
import os

from fastapi import FastAPI
from prometheus_client import CONTENT_TYPE_LATEST, generate_latest
from starlette.responses import Response

from server.routers import inference

logging.basicConfig(level=os.environ.get("LOG_LEVEL", "INFO"))
logger = logging.getLogger("llm-bench")

if os.environ.get("OTEL_EXPORTER_OTLP_ENDPOINT"):
    try:
        from traces.otel_setup import configure_tracing

        configure_tracing()
    except Exception as exc:  # pragma: no cover
        logger.warning("OpenTelemetry setup skipped: %s", exc)


def create_app() -> FastAPI:
    app = FastAPI(title="LLM Inference Benchmark API", version="0.1.0")
    app.include_router(inference.router)

    @app.get("/healthz")
    async def healthz() -> dict[str, str]:
        return {"status": "ok"}

    @app.get("/metrics")
    async def metrics() -> Response:
        data = generate_latest()
        return Response(content=data, media_type=CONTENT_TYPE_LATEST)

    return app


app = create_app()
