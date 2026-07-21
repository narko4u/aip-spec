#!/usr/bin/env python3
"""
AIP MCP Server — exposes AIP actions as MCP tools.

Registers tools that wrap the `aip` binary:
  - aip_keygen        Generate Ed25519 key pair
  - aip_negotiate     Start/continue a negotiation
  - aip_accept        Accept an offer
  - aip_decline       Decline an offer
  - aip_execute       Execute an action
  - aip_sign_contract Sign a binding contract
  - aip_settle        Settle a contract
  - aip_verify        Verify an evidence receipt
  - aip_demo          Run full end-to-end demo

Usage:
  python aip_mcp_server.py          # runs on stdio (for Hermes config)
  python aip_mcp_server.py --http   # runs on HTTP (for testing)
"""

import argparse
import json
import os
import subprocess
import sys
import tempfile
from pathlib import Path
from typing import Any

# ── MCP SDK ──────────────────────────────────────────────────────────
import anyio
from mcp.server import Server, NotificationOptions
from mcp.server.models import InitializationOptions
from mcp.types import (
    Tool,
    TextContent,
    CallToolResult,
    ErrorData,
)
import mcp.server.stdio

# ── AIP Bridge ──────────────────────────────────────────────────────

AIP_BINARY = os.environ.get("AIP_BINARY", "aip")


def _run_aip(*args: str, input_text: str | None = None) -> subprocess.CompletedProcess:
    """Run aip binary and return result."""
    return subprocess.run(
        [AIP_BINARY, *args],
        input=input_text,
        capture_output=True,
        text=True,
        timeout=30,
    )


def _last_json(text: str) -> dict[str, Any]:
    """Extract the last JSON object from CLI output."""
    text = text.strip()
    last_brace = text.rfind("{")
    if last_brace == -1:
        return {"raw_output": text}

    candidate = text[last_brace:]
    depth = 0
    end = -1
    for i, ch in enumerate(candidate):
        if ch == "{":
            depth += 1
        elif ch == "}":
            depth -= 1
            if depth == 0:
                end = i + 1
                break

    if end > 0:
        try:
            return json.loads(candidate[:end])
        except json.JSONDecodeError:
            pass
    return {"raw_output": text}


# ── Tool Implementations ────────────────────────────────────────────

async def tool_keygen(args: dict) -> dict:
    """Generate a new Ed25519 key pair."""
    proc = _run_aip("keygen")
    if proc.returncode != 0:
        return {"error": proc.stderr}

    keys = {}
    for line in proc.stdout.strip().split("\n"):
        if "Public Key:" in line:
            keys["public_key"] = line.split("Public Key:")[-1].strip()
        elif "Private Key:" in line:
            keys["private_key"] = line.split("Private Key:")[-1].strip().split(" ")[0]
    return keys


async def tool_negotiate(args: dict) -> dict:
    """Start or continue a negotiation session."""
    cmd = ["negotiate", "--role", args["role"], "--action", args["action_id"]]
    if args.get("session"):
        cmd.extend(["--session", args["session"]])
    if args.get("provider_key"):
        cmd.extend(["--provider-key", args["provider_key"]])
    if args.get("consumer_key"):
        cmd.extend(["--consumer-key", args["consumer_key"]])

    proc = _run_aip(*cmd)
    if proc.returncode != 0:
        return {"error": proc.stderr}
    return _last_json(proc.stdout)


async def tool_accept(args: dict) -> dict:
    """Accept an offer in a negotiation."""
    cmd = [
        "negotiate", "--accept",
        "--session", args["session"],
        "--role", args["role"],
    ]
    if args["role"] == "provider":
        cmd.extend(["--provider-key", args["key"]])
    else:
        cmd.extend(["--consumer-key", args["key"]])

    proc = _run_aip(*cmd)
    if proc.returncode != 0:
        return {"error": proc.stderr}
    return _last_json(proc.stdout)


async def tool_decline(args: dict) -> dict:
    """Decline an offer in a negotiation."""
    cmd = [
        "negotiate", "--decline",
        "--session", args["session"],
        "--role", args["role"],
    ]
    if args["role"] == "provider":
        cmd.extend(["--provider-key", args["key"]])
    else:
        cmd.extend(["--consumer-key", args["key"]])

    proc = _run_aip(*cmd)
    if proc.returncode != 0:
        return {"error": proc.stderr}
    return _last_json(proc.stdout)


async def tool_execute(args: dict) -> dict:
    """Execute an action against a schema.

    The schema is provided as a JSON string (inline schema).
    Optionally provide input_data as a JSON string.
    """
    # Write schema to temp file
    schema_data = args.get("schema", "")
    if isinstance(schema_data, str):
        schema_data = json.loads(schema_data)

    with tempfile.NamedTemporaryFile(mode="w", suffix=".json", delete=False) as f:
        json.dump(schema_data, f)
        schema_path = f.name

    input_path = None
    if args.get("input_data"):
        input_data = args["input_data"]
        if isinstance(input_data, str):
            input_data = json.loads(input_data)
        with tempfile.NamedTemporaryFile(mode="w", suffix=".json", delete=False) as f:
            json.dump(input_data, f)
            input_path = f.name

    try:
        cmd = ["execute", schema_path]
        if input_path:
            cmd.append(input_path)
        if args.get("witnessos_url"):
            cmd.extend(["--witnessos", args["witnessos_url"]])

        proc = _run_aip(*cmd)
        if proc.returncode != 0:
            return {"error": proc.stderr}
        return _last_json(proc.stdout)
    finally:
        os.unlink(schema_path)
        if input_path:
            os.unlink(input_path)


async def tool_sign_contract(args: dict) -> dict:
    """Sign a contract binding.

    Provide the binding as a JSON string or use default values.
    """
    binding_data = args.get("binding", {})
    if isinstance(binding_data, str):
        binding_data = json.loads(binding_data)

    with tempfile.NamedTemporaryFile(mode="w", suffix=".json", delete=False) as f:
        json.dump(binding_data, f)
        binding_path = f.name

    try:
        proc = _run_aip(
            "sign-contract",
            "--binding", binding_path,
            "--key", args["key"],
            "--party", args["party"],
        )
        if proc.returncode != 0:
            return {"error": proc.stderr}
        return _last_json(proc.stdout)
    finally:
        os.unlink(binding_path)


async def tool_settle(args: dict) -> dict:
    """Record a settlement for a contract.

    Provide the contract binding as a JSON string.
    """
    binding_data = args.get("binding", {})
    if isinstance(binding_data, str):
        binding_data = json.loads(binding_data)

    with tempfile.NamedTemporaryFile(mode="w", suffix=".json", delete=False) as f:
        json.dump(binding_data, f)
        contract_path = f.name

    try:
        proc = _run_aip("settle", contract_path)
        if proc.returncode != 0:
            return {"error": proc.stderr}
        return _last_json(proc.stdout)
    finally:
        os.unlink(contract_path)


async def tool_verify(args: dict) -> dict:
    """Verify an evidence receipt signature."""
    receipt_data = args.get("receipt", {})
    if isinstance(receipt_data, str):
        receipt_data = json.loads(receipt_data)

    with tempfile.NamedTemporaryFile(mode="w", suffix=".json", delete=False) as f:
        json.dump(receipt_data, f)
        receipt_path = f.name

    try:
        proc = _run_aip("verify", receipt_path, args["public_key"])
        if proc.returncode != 0:
            return {"valid": False, "error": proc.stderr.strip()}
        return {"valid": True, "output": proc.stdout.strip()}
    finally:
        os.unlink(receipt_path)


async def tool_demo(args: dict) -> dict:
    """Run the full end-to-end AIP demo."""
    proc = _run_aip("demo")
    if proc.returncode != 0:
        return {"error": proc.stderr}
    return {"demo_output": proc.stdout}


# ── Tool Definitions ────────────────────────────────────────────────

TOOLS: dict[str, tuple[Tool, Any]] = {
    "aip_keygen": (
        Tool(
            name="aip_keygen",
            description="Generate a new Ed25519 key pair for AIP operations",
            inputSchema={
                "type": "object",
                "properties": {},
            },
        ),
        tool_keygen,
    ),
    "aip_negotiate": (
        Tool(
            name="aip_negotiate",
            description="Start or continue a negotiation session between two agents",
            inputSchema={
                "type": "object",
                "required": ["role", "action_id"],
                "properties": {
                    "role": {
                        "type": "string",
                        "enum": ["provider", "consumer"],
                        "description": "Your role in the negotiation",
                    },
                    "action_id": {
                        "type": "string",
                        "description": "Action schema ID to negotiate",
                    },
                    "session": {
                        "type": "string",
                        "description": "Existing session ID (omit for new session)",
                    },
                    "provider_key": {
                        "type": "string",
                        "description": "Provider's hex-encoded private key",
                    },
                    "consumer_key": {
                        "type": "string",
                        "description": "Consumer's hex-encoded private key",
                    },
                },
            },
        ),
        tool_negotiate,
    ),
    "aip_accept": (
        Tool(
            name="aip_accept",
            description="Accept an offer in a negotiation session",
            inputSchema={
                "type": "object",
                "required": ["session", "role", "key"],
                "properties": {
                    "session": {"type": "string", "description": "Negotiation session ID"},
                    "role": {"type": "string", "enum": ["provider", "consumer"]},
                    "key": {"type": "string", "description": "Your hex-encoded private key"},
                },
            },
        ),
        tool_accept,
    ),
    "aip_decline": (
        Tool(
            name="aip_decline",
            description="Decline an offer in a negotiation session",
            inputSchema={
                "type": "object",
                "required": ["session", "role", "key"],
                "properties": {
                    "session": {"type": "string", "description": "Negotiation session ID"},
                    "role": {"type": "string", "enum": ["provider", "consumer"]},
                    "key": {"type": "string", "description": "Your hex-encoded private key"},
                },
            },
        ),
        tool_decline,
    ),
    "aip_execute": (
        Tool(
            name="aip_execute",
            description="Execute an action against an AIP action schema",
            inputSchema={
                "type": "object",
                "required": ["schema"],
                "properties": {
                    "schema": {
                        "type": "object",
                        "description": "Action schema JSON object (with action_id, input_schema, output_schema, transport)",
                    },
                    "input_data": {
                        "type": "object",
                        "description": "Optional input data to pass to the action",
                    },
                    "witnessos_url": {
                        "type": "string",
                        "description": "Optional WitnessOS URL to push evidence receipt",
                    },
                },
            },
        ),
        tool_execute,
    ),
    "aip_sign_contract": (
        Tool(
            name="aip_sign_contract",
            description="Sign a binding contract with your Ed25519 private key",
            inputSchema={
                "type": "object",
                "required": ["key", "party"],
                "properties": {
                    "key": {"type": "string", "description": "Your hex-encoded Ed25519 private key"},
                    "party": {"type": "string", "enum": ["provider", "consumer"]},
                    "binding": {
                        "type": "object",
                        "description": "Contract binding JSON (omit for defaults)",
                    },
                },
            },
        ),
        tool_sign_contract,
    ),
    "aip_settle": (
        Tool(
            name="aip_settle",
            description="Record a settlement for a contract",
            inputSchema={
                "type": "object",
                "properties": {
                    "binding": {
                        "type": "object",
                        "description": "Contract binding JSON to settle",
                    },
                },
            },
        ),
        tool_settle,
    ),
    "aip_verify": (
        Tool(
            name="aip_verify",
            description="Verify an evidence receipt signature",
            inputSchema={
                "type": "object",
                "required": ["receipt", "public_key"],
                "properties": {
                    "receipt": {
                        "type": "object",
                        "description": "Evidence receipt JSON",
                    },
                    "public_key": {
                        "type": "string",
                        "description": "Hex-encoded Ed25519 public key to verify against",
                    },
                },
            },
        ),
        tool_verify,
    ),
    "aip_demo": (
        Tool(
            name="aip_demo",
            description="Run the full end-to-end AIP demo (keygen → negotiate → execute → settle → verify)",
            inputSchema={
                "type": "object",
                "properties": {},
            },
        ),
        tool_demo,
    ),
}


# ── MCP Server ──────────────────────────────────────────────────────

server = Server("aip-mcp-server")


@server.list_tools()
async def list_tools() -> list[Tool]:
    return [tool for tool, _ in TOOLS.values()]


@server.call_tool()
async def call_tool(name: str, arguments: dict) -> list[TextContent]:
    if name not in TOOLS:
        return [TextContent(
            type="text",
            text=json.dumps({"error": f"Unknown tool: {name}"}),
        )]

    _, handler = TOOLS[name]
    try:
        result = await handler(arguments)
        return [TextContent(
            type="text",
            text=json.dumps(result, indent=2),
        )]
    except Exception as e:
        return [TextContent(
            type="text",
            text=json.dumps({"error": str(e)}),
        )]


async def main_stdio():
    """Run MCP server over stdio (for Hermes config)."""
    async with mcp.server.stdio.stdio_server() as (read, write):
        await server.run(
            read,
            write,
            InitializationOptions(
                server_name="aip-mcp-server",
                server_version="0.1.0",
                capabilities=server.get_capabilities(
                    notification_options=NotificationOptions(),
                    experimental_capabilities={},
                ),
            ),
        )


async def main_http(port: int = 8768):
    """Run MCP server over HTTP using uvicorn + Starlette + StreamableHTTP."""
    from mcp.server.streamable_http import StreamableHTTPServerTransport
    from starlette.applications import Starlette
    from starlette.routing import Mount
    from starlette.types import ASGIApp
    import uvicorn

    transport = StreamableHTTPServerTransport(mcp_session_id=None)
    routes = [Mount("/mcp", app=transport.handle_request)]
    app = Starlette(routes=routes)

    config = uvicorn.Config(app, host="0.0.0.0", port=port, log_level="info")
    server = uvicorn.Server(config)
    await server.serve()


async def main():
    parser = argparse.ArgumentParser(description="AIP MCP Server")
    parser.add_argument("--http", action="store_true", help="Run as HTTP server instead of stdio")
    parser.add_argument("--port", type=int, default=8768, help="HTTP port (default: 8768)")
    args = parser.parse_args()

    if args.http:
        await main_http(args.port)
    else:
        await main_stdio()


if __name__ == "__main__":
    anyio.run(main)
