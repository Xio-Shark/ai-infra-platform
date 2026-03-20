from __future__ import annotations

import os
import uuid
from typing import Any, AsyncIterator

import httpx

from server.schemas.request_response import (
    ChatCompletionChoice,
    ChatCompletionChunk,
    ChatCompletionRequest,
    ChatCompletionResponse,
    ChatChunkChoice,
    ChatDelta,
    ChatMessage,
)


def _upstream_base() -> str | None:
    raw = os.environ.get("VLLM_BASE_URL") or os.environ.get("UPSTREAM_BASE_URL", "")
    return raw.rstrip("/") or None


async def forward_chat_completion(
    body: dict[str, Any],
) -> tuple[ChatCompletionResponse | None, httpx.Response | None]:
    base = _upstream_base()
    if not base:
        return None, None
    url = f"{base}/chat/completions"
    async with httpx.AsyncClient(timeout=httpx.Timeout(600.0)) as client:
        r = await client.post(url, json=body)
    if r.status_code >= 400:
        return None, r
    data = r.json()
    return ChatCompletionResponse.model_validate(data), r


async def forward_chat_completion_stream(
    body: dict[str, Any],
) -> AsyncIterator[bytes]:
    base = _upstream_base()
    if not base:
        return
    url = f"{base}/chat/completions"
    body = {**body, "stream": True}
    async with httpx.AsyncClient(timeout=httpx.Timeout(600.0)) as client:
        async with client.stream("POST", url, json=body) as resp:
            resp.raise_for_status()
            async for chunk in resp.aiter_bytes():
                yield chunk


def mock_completion(req: ChatCompletionRequest) -> ChatCompletionResponse:
    n = min(req.max_tokens or 32, 64)
    text = "ok " * max(1, n // 3)
    return ChatCompletionResponse(
        id=f"chatcmpl-{uuid.uuid4().hex[:12]}",
        model=req.model,
        choices=[
            ChatCompletionChoice(
                index=0,
                message=ChatMessage(role="assistant", content=text[: n * 4]),
                finish_reason="stop",
            )
        ],
        usage={"prompt_tokens": 10, "completion_tokens": n, "total_tokens": 10 + n},
    )


async def mock_stream(req: ChatCompletionRequest) -> AsyncIterator[str]:
    cid = f"chatcmpl-{uuid.uuid4().hex[:12]}"
    first = ChatCompletionChunk(
        id=cid,
        model=req.model,
        choices=[
            ChatChunkChoice(index=0, delta=ChatDelta(role="assistant", content=""))
        ],
    )
    yield first.sse_line()
    await __import__("asyncio").sleep(0.01)
    words = ["mock", "token", "stream", "done"]
    for w in words:
        chunk = ChatCompletionChunk(
            id=cid,
            model=req.model,
            choices=[ChatChunkChoice(index=0, delta=ChatDelta(content=w + " "))],
        )
        yield chunk.sse_line()
        await __import__("asyncio").sleep(0.005)
    end = ChatCompletionChunk(
        id=cid,
        model=req.model,
        choices=[ChatChunkChoice(index=0, delta=ChatDelta(), finish_reason="stop")],
    )
    yield end.sse_line()
    yield "data: [DONE]\n\n"
