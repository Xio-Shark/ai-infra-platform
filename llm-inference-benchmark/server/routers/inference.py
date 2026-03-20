from __future__ import annotations

import json
import time
from typing import Any

from fastapi import APIRouter, HTTPException, Response
from fastapi.responses import JSONResponse, StreamingResponse
from prometheus_client import Counter, Histogram

from server.schemas.request_response import ChatCompletionRequest, ErrorResponse
from server.services import engine_client

router = APIRouter(prefix="/v1", tags=["openai"])

REQUESTS = Counter(
    "llm_bench_inference_requests_total",
    "Inference requests",
    ["endpoint", "status"],
)
LATENCY = Histogram(
    "llm_bench_inference_latency_seconds",
    "End-to-end handler latency (non-stream)",
    ["endpoint"],
    buckets=(0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 30, 60, 120),
)


@router.post("/chat/completions")
async def chat_completions(raw: dict[str, Any]) -> Response:
    t0 = time.perf_counter()
    try:
        req = ChatCompletionRequest.model_validate(raw)
    except Exception as e:
        REQUESTS.labels(endpoint="chat_completions", status="4xx").inc()
        return JSONResponse(
            status_code=400,
            content=ErrorResponse(
                error={"message": str(e), "type": "invalid_request"}
            ).model_dump(),
        )

    body = req.model_dump(exclude_none=True)

    if req.stream:
        if engine_client._upstream_base():

            async def upstream_iter():
                try:
                    async for chunk in engine_client.forward_chat_completion_stream(
                        body
                    ):
                        yield chunk
                except Exception as ex:
                    err = json.dumps(
                        {"error": {"message": str(ex), "type": "upstream_error"}}
                    )
                    yield f"data: {err}\n\n".encode()

            REQUESTS.labels(endpoint="chat_completions", status="stream").inc()
            return StreamingResponse(
                upstream_iter(),
                media_type="text/event-stream",
            )

        async def mock_iter():
            async for line in engine_client.mock_stream(req):
                yield line.encode()

        REQUESTS.labels(endpoint="chat_completions", status="stream_mock").inc()
        return StreamingResponse(mock_iter(), media_type="text/event-stream")

    parsed, upstream_resp = await engine_client.forward_chat_completion(body)
    if parsed is not None:
        REQUESTS.labels(endpoint="chat_completions", status="2xx").inc()
        LATENCY.labels(endpoint="chat_completions").observe(time.perf_counter() - t0)
        return JSONResponse(content=parsed.model_dump())

    if upstream_resp is not None:
        code = upstream_resp.status_code
        bucket = "5xx" if code >= 500 else "4xx" if code >= 400 else "2xx"
        REQUESTS.labels(endpoint="chat_completions", status=bucket).inc()
        try:
            detail = upstream_resp.json()
        except Exception:
            detail = {"error": upstream_resp.text}
        raise HTTPException(status_code=upstream_resp.status_code, detail=detail)

    out = engine_client.mock_completion(req)
    REQUESTS.labels(endpoint="chat_completions", status="2xx_mock").inc()
    LATENCY.labels(endpoint="chat_completions").observe(time.perf_counter() - t0)
    return JSONResponse(content=out.model_dump())
