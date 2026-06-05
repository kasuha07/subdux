#!/usr/bin/env python3
"""Plan and apply Subdux account settings changes.

This script manages supporting setup data only: categories, currencies,
payment methods, and preferred currency. It intentionally does not manage
subscriptions.
"""

from __future__ import annotations

import argparse
import json
import os
import stat
import sys
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass
from pathlib import Path
from typing import Any


DEFAULT_BASE_URL = "http://127.0.0.1:8080"
DEFAULT_CONFIG_PATH = Path("~/.config/subdux/config.yaml").expanduser()


class SubduxError(RuntimeError):
    pass


@dataclass
class Config:
    base_url: str
    api_key: str
    timeout_seconds: float
    config_path: Path


def load_json(path: Path) -> Any:
    try:
        with path.open("r", encoding="utf-8") as handle:
            return json.load(handle)
    except FileNotFoundError as exc:
        raise SubduxError(f"file not found: {path}") from exc
    except json.JSONDecodeError as exc:
        raise SubduxError(f"invalid JSON in {path}: {exc}") from exc


def write_json(path: Path, data: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8") as handle:
        json.dump(data, handle, indent=2, ensure_ascii=False, sort_keys=True)
        handle.write("\n")


def parse_scalar(value: str) -> Any:
    value = value.strip()
    if value == "" or value == "null" or value == "~":
        return None
    if value == "true":
        return True
    if value == "false":
        return False
    if (value.startswith('"') and value.endswith('"')) or (value.startswith("'") and value.endswith("'")):
        try:
            return json.loads(value) if value.startswith('"') else value[1:-1].replace("''", "'")
        except json.JSONDecodeError as exc:
            raise SubduxError(f"invalid quoted config value: {value}") from exc
    try:
        if "." in value:
            return float(value)
        return int(value)
    except ValueError:
        return value


def load_simple_yaml_mapping(path: Path) -> dict[str, Any]:
    data: dict[str, Any] = {}
    try:
        lines = path.read_text(encoding="utf-8").splitlines()
    except FileNotFoundError as exc:
        raise SubduxError(f"file not found: {path}") from exc

    for line_number, raw_line in enumerate(lines, start=1):
        stripped = raw_line.strip()
        if not stripped or stripped.startswith("#"):
            continue
        if raw_line[: len(raw_line) - len(raw_line.lstrip())]:
            raise SubduxError(f"config only supports top-level key/value pairs: {path}:{line_number}")
        if ":" not in raw_line:
            raise SubduxError(f"invalid config line, expected key: value: {path}:{line_number}")
        key, value = raw_line.split(":", 1)
        key = key.strip()
        if not key:
            raise SubduxError(f"empty config key: {path}:{line_number}")
        data[key] = parse_scalar(value)
    return data


def dump_simple_yaml_mapping(data: dict[str, Any]) -> str:
    lines = []
    for key in sorted(data):
        value = data[key]
        if value is None:
            rendered = "null"
        elif isinstance(value, bool):
            rendered = "true" if value else "false"
        elif isinstance(value, (int, float)):
            rendered = str(value)
        else:
            rendered = json.dumps(str(value), ensure_ascii=False)
        lines.append(f"{key}: {rendered}")
    return "\n".join(lines) + "\n"


def write_yaml_mapping(path: Path, data: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(dump_simple_yaml_mapping(data), encoding="utf-8")


def resolve_config_path(args: argparse.Namespace) -> Path:
    raw = args.config or os.environ.get("SUBDUX_CONFIG")
    if raw:
        return Path(raw).expanduser()
    return DEFAULT_CONFIG_PATH


def read_config_file(path: Path) -> dict[str, Any]:
    if not path.exists():
        return {}
    suffix = path.suffix.lower()
    if suffix in (".yaml", ".yml", ""):
        return load_simple_yaml_mapping(path)
    if suffix == ".json":
        data = load_json(path)
        if not isinstance(data, dict):
            raise SubduxError(f"config file must contain a JSON object: {path}")
        return data
    raise SubduxError(f"unsupported config file extension: {path.suffix}. Use .yaml, .yml, or .json.")


def check_secret_file_mode(path: Path, config: dict[str, Any]) -> None:
    if not config.get("api_key"):
        return
    if os.name == "nt":
        return
    mode = stat.S_IMODE(path.stat().st_mode)
    if mode & 0o077:
        raise SubduxError(
            f"config file {path} contains api_key and must not be readable by group or others; run chmod 600 {path}"
        )


def resolve_config(args: argparse.Namespace, require_api_key: bool = True) -> Config:
    config_path = resolve_config_path(args)
    file_config = read_config_file(config_path)
    if config_path.exists():
        check_secret_file_mode(config_path, file_config)

    base_url = (
        args.base_url
        or os.environ.get("SUBDUX_BASE_URL")
        or str(file_config.get("base_url") or "")
        or DEFAULT_BASE_URL
    )
    timeout = args.timeout_seconds
    if timeout is None:
        timeout = os.environ.get("SUBDUX_TIMEOUT_SECONDS") or file_config.get("timeout_seconds") or 15
    try:
        timeout_seconds = float(timeout)
    except (TypeError, ValueError) as exc:
        raise SubduxError("timeout must be a number") from exc
    if timeout_seconds <= 0:
        raise SubduxError("timeout must be greater than zero")

    api_key_env = str(file_config.get("api_key_env") or "SUBDUX_API_KEY")
    api_key = (
        args.api_key
        or os.environ.get("SUBDUX_API_KEY")
        or os.environ.get(api_key_env)
        or str(file_config.get("api_key") or "")
    )
    api_key = api_key.strip()
    if require_api_key and not api_key:
        raise SubduxError(
            "Subdux API key is required. Set SUBDUX_API_KEY, pass --api-key, or configure api_key_env."
        )

    return Config(
        base_url=base_url.rstrip("/"),
        api_key=api_key,
        timeout_seconds=timeout_seconds,
        config_path=config_path,
    )


class SubduxClient:
    def __init__(self, config: Config) -> None:
        self.config = config

    def request(self, method: str, path: str, payload: Any | None = None) -> Any:
        url = self.config.base_url + path
        body = None
        headers = {
            "Accept": "application/json",
            "X-API-Key": self.config.api_key,
        }
        if payload is not None:
            body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
            headers["Content-Type"] = "application/json"

        req = urllib.request.Request(url, data=body, method=method.upper(), headers=headers)
        try:
            with urllib.request.urlopen(req, timeout=self.config.timeout_seconds) as resp:
                content = resp.read()
                if not content:
                    return None
                return json.loads(content.decode("utf-8"))
        except urllib.error.HTTPError as exc:
            message = exc.read().decode("utf-8", errors="replace")
            try:
                parsed = json.loads(message)
                if isinstance(parsed, dict) and parsed.get("error"):
                    message = str(parsed["error"])
            except json.JSONDecodeError:
                pass
            raise SubduxError(f"{method.upper()} {path} failed with HTTP {exc.code}: {message}") from exc
        except urllib.error.URLError as exc:
            raise SubduxError(f"failed to connect to {url}: {exc.reason}") from exc

    def list_state(self) -> dict[str, Any]:
        return {
            "preferred_currency": self.request("GET", "/api/preferences/currency").get("preferred_currency"),
            "currencies": self.request("GET", "/api/currencies"),
            "categories": self.request("GET", "/api/categories"),
            "payment_methods": self.request("GET", "/api/payment-methods"),
        }


def normalized_name(value: str) -> str:
    return " ".join(value.strip().casefold().split())


def normalize_currency_code(value: str) -> str:
    return value.strip().upper()


def require_list(data: dict[str, Any], key: str) -> list[dict[str, Any]]:
    if key not in data:
        return []
    value = data[key]
    if not isinstance(value, list):
        raise SubduxError(f"desired field {key!r} must be an array")
    for index, item in enumerate(value):
        if not isinstance(item, dict):
            raise SubduxError(f"desired field {key}[{index}] must be an object")
    return value


def validate_desired(desired: Any) -> dict[str, Any]:
    if not isinstance(desired, dict):
        raise SubduxError("desired file must contain a JSON object")

    result: dict[str, Any] = {}
    if "preferred_currency" in desired:
        if not isinstance(desired["preferred_currency"], str):
            raise SubduxError("preferred_currency must be a string")
        result["preferred_currency"] = normalize_currency_code(desired["preferred_currency"])

    for key in ("currencies", "categories", "payment_methods"):
        result[key] = require_list(desired, key)
    return result


def next_sort_order(items: list[dict[str, Any]], field: str) -> int:
    if not items:
        return 0
    return max(int(item.get(field, 0)) for item in items) + 1


def build_plan(current: dict[str, Any], desired: dict[str, Any]) -> dict[str, Any]:
    actions: list[dict[str, Any]] = []

    existing_currencies = {
        normalize_currency_code(str(item.get("code", ""))): item
        for item in current.get("currencies", [])
        if item.get("code")
    }
    currency_sort = next_sort_order(current.get("currencies", []), "sort_order")
    reorder_currencies: list[dict[str, Any]] = []
    for index, item in enumerate(desired.get("currencies", [])):
        code = normalize_currency_code(str(item.get("code", "")))
        if not code:
            raise SubduxError(f"currencies[{index}].code is required")
        target = {
            "code": code,
            "symbol": str(item.get("symbol", "")).strip(),
            "alias": str(item.get("alias", "")).strip(),
            "sort_order": index,
        }
        existing = existing_currencies.get(code)
        if existing is None:
            actions.append({"type": "create_currency", "payload": target})
        else:
            update: dict[str, Any] = {}
            for field in ("symbol", "alias"):
                if target[field] != str(existing.get(field, "")):
                    update[field] = target[field]
            if int(existing.get("sort_order", currency_sort)) != index:
                reorder_currencies.append({"id": existing["id"], "sort_order": index})
            if update:
                actions.append({"type": "update_currency", "id": existing["id"], "code": code, "payload": update})
    if reorder_currencies:
        actions.append({"type": "reorder_currencies", "payload": reorder_currencies})

    existing_categories = {
        normalized_name(str(item.get("name", ""))): item
        for item in current.get("categories", [])
        if item.get("name")
    }
    reorder_categories: list[dict[str, Any]] = []
    for index, item in enumerate(desired.get("categories", [])):
        name = str(item.get("name", "")).strip()
        if not name:
            raise SubduxError(f"categories[{index}].name is required")
        existing = existing_categories.get(normalized_name(name))
        if existing is None:
            actions.append({"type": "create_category", "payload": {"name": name, "display_order": index}})
        else:
            payload: dict[str, Any] = {}
            if str(existing.get("name", "")) != name:
                payload["name"] = name
            if int(existing.get("display_order", 0)) != index:
                reorder_categories.append({"id": existing["id"], "sort_order": index})
            if payload:
                actions.append({"type": "update_category", "id": existing["id"], "name": existing.get("name"), "payload": payload})
    if reorder_categories:
        actions.append({"type": "reorder_categories", "payload": reorder_categories})

    existing_methods = {
        normalized_name(str(item.get("name", ""))): item
        for item in current.get("payment_methods", [])
        if item.get("name")
    }
    reorder_methods: list[dict[str, Any]] = []
    for index, item in enumerate(desired.get("payment_methods", [])):
        name = str(item.get("name", "")).strip()
        if not name:
            raise SubduxError(f"payment_methods[{index}].name is required")
        icon = str(item.get("icon", "")).strip()
        existing = existing_methods.get(normalized_name(name))
        if existing is None:
            actions.append({"type": "create_payment_method", "payload": {"name": name, "icon": icon, "sort_order": index}})
        else:
            payload = {}
            if str(existing.get("name", "")) != name:
                payload["name"] = name
            if icon != str(existing.get("icon", "")):
                payload["icon"] = icon
            if int(existing.get("sort_order", 0)) != index:
                reorder_methods.append({"id": existing["id"], "sort_order": index})
            if payload:
                actions.append({"type": "update_payment_method", "id": existing["id"], "name": existing.get("name"), "payload": payload})
    if reorder_methods:
        actions.append({"type": "reorder_payment_methods", "payload": reorder_methods})

    preferred = desired.get("preferred_currency")
    current_preferred = normalize_currency_code(str(current.get("preferred_currency") or ""))
    if preferred and preferred != current_preferred:
        actions.append({"type": "update_preferred_currency", "payload": {"preferred_currency": preferred}})

    return {
        "version": 1,
        "summary": summarize_actions(actions),
        "actions": actions,
    }


def build_cleanup_plan(current: dict[str, Any]) -> dict[str, Any]:
    subscriptions = current.get("subscriptions")
    if not isinstance(subscriptions, list):
        raise SubduxError("cleanup requires subscriptions in current state")

    used_currency_codes = {
        normalize_currency_code(str(item.get("currency", "")))
        for item in subscriptions
        if isinstance(item, dict) and item.get("currency")
    }
    used_category_ids = {
        int(item["category_id"])
        for item in subscriptions
        if isinstance(item, dict) and isinstance(item.get("category_id"), int)
    }
    used_category_names = {
        normalized_name(str(item.get("category", "")))
        for item in subscriptions
        if isinstance(item, dict) and item.get("category")
    }
    used_payment_method_ids = {
        int(item["payment_method_id"])
        for item in subscriptions
        if isinstance(item, dict) and isinstance(item.get("payment_method_id"), int)
    }

    preferred = normalize_currency_code(str(current.get("preferred_currency") or ""))
    actions: list[dict[str, Any]] = []

    for currency in current.get("currencies", []):
        code = normalize_currency_code(str(currency.get("code", "")))
        if code and code != preferred and code not in used_currency_codes:
            actions.append({"type": "delete_currency", "id": currency["id"], "code": code})

    for category in current.get("categories", []):
        category_id = int(category["id"])
        name = normalized_name(str(category.get("name", "")))
        if category_id not in used_category_ids and name not in used_category_names:
            actions.append({"type": "delete_category", "id": category_id, "name": category.get("name")})

    for method in current.get("payment_methods", []):
        method_id = int(method["id"])
        if method_id not in used_payment_method_ids:
            actions.append({"type": "delete_payment_method", "id": method_id, "name": method.get("name")})

    return {
        "version": 1,
        "summary": summarize_actions(actions),
        "actions": actions,
    }


def summarize_actions(actions: list[dict[str, Any]]) -> dict[str, int]:
    summary: dict[str, int] = {}
    for action in actions:
        key = str(action["type"])
        summary[key] = summary.get(key, 0) + 1
    return summary


def apply_action(client: SubduxClient, action: dict[str, Any]) -> Any:
    action_type = action.get("type")
    payload = action.get("payload")
    if action_type == "create_currency":
        return client.request("POST", "/api/currencies", payload)
    if action_type == "update_currency":
        return client.request("PUT", f"/api/currencies/{action['id']}", payload)
    if action_type == "reorder_currencies":
        return client.request("PUT", "/api/currencies/reorder", payload)
    if action_type == "delete_currency":
        return client.request("DELETE", f"/api/currencies/{action['id']}")
    if action_type == "create_category":
        return client.request("POST", "/api/categories", payload)
    if action_type == "update_category":
        return client.request("PUT", f"/api/categories/{action['id']}", payload)
    if action_type == "reorder_categories":
        return client.request("PUT", "/api/categories/reorder", payload)
    if action_type == "delete_category":
        return client.request("DELETE", f"/api/categories/{action['id']}")
    if action_type == "create_payment_method":
        return client.request("POST", "/api/payment-methods", payload)
    if action_type == "update_payment_method":
        return client.request("PUT", f"/api/payment-methods/{action['id']}", payload)
    if action_type == "reorder_payment_methods":
        return client.request("PUT", "/api/payment-methods/reorder", payload)
    if action_type == "delete_payment_method":
        return client.request("DELETE", f"/api/payment-methods/{action['id']}")
    if action_type == "update_preferred_currency":
        return client.request("PUT", "/api/preferences/currency", payload)
    raise SubduxError(f"unknown action type: {action_type}")


def print_output(data: Any, as_json: bool) -> None:
    if as_json:
        print(json.dumps(data, indent=2, ensure_ascii=False, sort_keys=True))
        return
    if isinstance(data, dict) and "summary" in data and "actions" in data:
        print("Plan summary:")
        if data["summary"]:
            for key, count in sorted(data["summary"].items()):
                print(f"- {key}: {count}")
        else:
            print("- no changes")
        print(f"Actions: {len(data['actions'])}")
        return
    print(json.dumps(data, indent=2, ensure_ascii=False, sort_keys=True))


def cmd_list(args: argparse.Namespace) -> int:
    config = resolve_config(args)
    client = SubduxClient(config)
    print_output(client.list_state(), args.json)
    return 0


def cmd_plan(args: argparse.Namespace) -> int:
    config = resolve_config(args)
    client = SubduxClient(config)
    desired = validate_desired(load_json(Path(args.desired)))
    current = client.list_state()
    plan = build_plan(current, desired)
    if args.out:
        write_json(Path(args.out), plan)
    print_output(plan, args.json)
    return 0


def cmd_cleanup_unused(args: argparse.Namespace) -> int:
    config = resolve_config(args)
    client = SubduxClient(config)
    current = client.list_state()
    subscriptions_payload = client.request("GET", "/api/subscriptions")
    if isinstance(subscriptions_payload, dict) and isinstance(subscriptions_payload.get("subscriptions"), list):
        current["subscriptions"] = subscriptions_payload["subscriptions"]
    elif isinstance(subscriptions_payload, list):
        current["subscriptions"] = subscriptions_payload
    else:
        raise SubduxError("unexpected /api/subscriptions response shape")
    plan = build_cleanup_plan(current)
    if args.out:
        write_json(Path(args.out), plan)
    print_output(plan, args.json)
    return 0


def cmd_apply(args: argparse.Namespace) -> int:
    config = resolve_config(args)
    client = SubduxClient(config)
    plan = load_json(Path(args.plan))
    if not isinstance(plan, dict) or not isinstance(plan.get("actions"), list):
        raise SubduxError("plan file must contain an actions array")
    results = []
    for index, action in enumerate(plan["actions"]):
        if not isinstance(action, dict):
            raise SubduxError(f"plan action {index} must be an object")
        results.append({"action": action.get("type"), "result": apply_action(client, action)})
    print_output({"applied": len(results), "results": results}, args.json)
    return 0


def cmd_write_config(args: argparse.Namespace) -> int:
    path = resolve_config_path(args)
    data: dict[str, Any] = {}
    if path.exists():
        data = read_config_file(path)
    data["base_url"] = args.base_url or os.environ.get("SUBDUX_BASE_URL") or data.get("base_url") or DEFAULT_BASE_URL
    data["api_key_env"] = args.api_key_env or data.get("api_key_env") or "SUBDUX_API_KEY"
    data["timeout_seconds"] = args.timeout_seconds or data.get("timeout_seconds") or 15
    data.setdefault("api_key", None)
    if path.suffix.lower() == ".json":
        write_json(path, data)
    else:
        write_yaml_mapping(path, data)
    if os.name != "nt":
        os.chmod(path, 0o600)
    print(f"Wrote {path}")
    return 0


def add_common_flags(parser: argparse.ArgumentParser) -> None:
    parser.add_argument("--config", help="Config file path. Defaults to SUBDUX_CONFIG or ~/.config/subdux/config.yaml.")
    parser.add_argument("--base-url", help="Subdux base URL. Overrides SUBDUX_BASE_URL and config.")
    parser.add_argument("--api-key", help="Subdux API key. Prefer SUBDUX_API_KEY to avoid shell history.")
    parser.add_argument("--timeout-seconds", type=float, help="HTTP timeout in seconds.")
    parser.add_argument("--json", action="store_true", help="Print machine-readable JSON output.")


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Manage Subdux account setup settings.")
    subparsers = parser.add_subparsers(dest="command", required=True)

    list_parser = subparsers.add_parser("list", help="List current settings.")
    add_common_flags(list_parser)
    list_parser.set_defaults(func=cmd_list)

    plan_parser = subparsers.add_parser("plan", help="Create a settings change plan from desired JSON.")
    add_common_flags(plan_parser)
    plan_parser.add_argument("--desired", required=True, help="Desired settings JSON file.")
    plan_parser.add_argument("--out", help="Write plan JSON to this path.")
    plan_parser.set_defaults(func=cmd_plan)

    cleanup_parser = subparsers.add_parser("cleanup-unused", help="Plan deletion of currencies, categories, and payment methods unused by subscriptions.")
    add_common_flags(cleanup_parser)
    cleanup_parser.add_argument("--out", help="Write cleanup plan JSON to this path.")
    cleanup_parser.set_defaults(func=cmd_cleanup_unused)

    apply_parser = subparsers.add_parser("apply", help="Apply a previously reviewed plan.")
    add_common_flags(apply_parser)
    apply_parser.add_argument("--plan", required=True, help="Plan JSON file.")
    apply_parser.set_defaults(func=cmd_apply)

    config_parser = subparsers.add_parser("write-config", help="Write local connection config without embedding secrets.")
    config_parser.add_argument("--config", help="Config file path. Defaults to SUBDUX_CONFIG or ~/.config/subdux/config.yaml.")
    config_parser.add_argument("--base-url", help="Subdux base URL.")
    config_parser.add_argument("--api-key-env", default="SUBDUX_API_KEY", help="Environment variable name that stores the API key.")
    config_parser.add_argument("--timeout-seconds", type=float, help="HTTP timeout in seconds.")
    config_parser.set_defaults(func=cmd_write_config)
    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    try:
        return args.func(args)
    except SubduxError as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
