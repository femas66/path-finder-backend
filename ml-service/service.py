"""Internal offline inference HTTP boundary; launch with the ML environment."""
from __future__ import annotations

import hashlib
import hmac
import json
import logging
import os
from pathlib import Path
import socket
import sys
import threading
from dataclasses import fields
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

ROOT = Path(__file__).resolve().parents[2] / "pathfinder-ml"
os.environ["HF_HUB_OFFLINE"] = "1"
os.environ["TRANSFORMERS_OFFLINE"] = "1"
sys.path.insert(0, str(Path(os.environ.get("ML_PIPELINE_SRC", ROOT / "src")).expanduser().resolve()))
from pipeline.schemas import CandidateInput, Job, RecommendationResponse, NerStatus

BODY_CAP = 1024 * 1024
ENCODER = "intfloat/multilingual-e5-small"
REVISION = "614241f622f53c4eeff9890bdc4f31cfecc418b3"
RANKER = "gb-e5-reviewed-dev-v1"


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError("duplicate field")
        result[key] = value
    return result


def bounded(value, limit=2000):
    if not isinstance(value, str) or len(value) > limit:
        raise ValueError("text bound")


def entity(cls, value):
    if not isinstance(value, dict) or set(value) - {f.name for f in fields(cls)}:
        raise ValueError("fields")
    return cls.from_dict(value)


def parse_payload(raw):
    data = json.loads(raw, object_pairs_hook=unique_object,
                      parse_constant=lambda _: (_ for _ in ()).throw(ValueError("constant")))
    if not isinstance(data, dict) or set(data) != {"candidate", "jobs"}:
        raise ValueError("wrapper")
    candidate = entity(CandidateInput, data["candidate"])
    bounded(candidate.student_id, 256)
    bounded(candidate.narrative)
    if candidate.cv_text is not None:
        bounded(candidate.cv_text, 100000)
    if candidate.consent_at is not None:
        bounded(candidate.consent_at, 128)
    for values in (candidate.manual_profile or {}).values():
        if len(values) > 200:
            raise ValueError("list bound")
        for value in values:
            bounded(value)
    if not isinstance(data["jobs"], list) or len(data["jobs"]) > 200:
        raise ValueError("catalog bound")
    jobs = [entity(Job, value) for value in data["jobs"]]
    if len({job.job_id for job in jobs}) != len(jobs):
        raise ValueError("duplicate job")
    for job in jobs:
        for name, value in job.to_dict().items():
            if isinstance(value, str):
                bounded(value, 20000 if name in {"description", "requirements_text"} else 2000)
            elif isinstance(value, list):
                if len(value) > 200:
                    raise ValueError("list bound")
                for text in value:
                    bounded(text)
    return candidate, jobs


class CheckedNer:
    """Remember errors the upstream orchestrator deliberately swallows."""
    def __init__(self, inner):
        self.inner = inner
        self.failed = False

    def __getattr__(self, name):
        return getattr(self.inner, name)

    def extract(self, *args, **kwargs):
        try:
            output = self.inner.extract(*args, **kwargs)
            if output.status is not NerStatus.OK:
                raise RuntimeError("NER unavailable")
            return output
        except Exception:
            self.failed = True
            raise


class State:
    def __init__(self, token, model=None):
        if not token or not token.isascii() or token.strip() != token:
            raise ValueError("ML_SERVICE_TOKEN required; use nonempty ASCII without surrounding whitespace")
        self.token = token.encode("ascii")
        self.model = model
        self.worker = threading.BoundedSemaphore(1)
        self.catalog_hash = None
        if model is not None:
            model.ner = CheckedNer(model.ner)

    def recommend(self, candidate, jobs):
        # No public clear API: register_job_vectors only updates this ID-keyed dict.
        digest = hashlib.sha256(json.dumps([j.to_dict() for j in jobs], sort_keys=True,
                                           ensure_ascii=True).encode()).digest()
        if digest != self.catalog_hash:
            self.model.matcher._job_vectors.clear()
            self.catalog_hash = digest
        self.model.ner.failed = False
        result = self.model.recommend(candidate, jobs)
        if self.model.ner.failed:
            raise RuntimeError("NER failure")
        if not isinstance(result, RecommendationResponse):
            raise RuntimeError("invalid model response")
        result.validate()
        if result.top_k != 3 or len(result.items) > 3:
            raise RuntimeError("invalid top_k")
        return result.to_dict()


def load_model():
    from pipeline.local_inference import build_local_recommender
    import torch
    threads = int(os.environ.get("ML_THREADS", "1"))
    if not 1 <= threads <= 64:
        raise ValueError("ML_THREADS must be 1..64")
    torch.set_num_threads(threads)
    directory = Path(os.environ["ML_TRAINING_DIR"]).expanduser().resolve()
    report = json.loads((directory / "report.json").read_text(), object_pairs_hook=unique_object)
    encoder = report["provenance"]["encoder_config"]
    if (report.get("selected_family") != "gb"
            or report["bundles"]["gb"]["model_version"] != RANKER
            or encoder["model_name"] != ENCODER or encoder["revision"] != REVISION):
        raise ValueError("unapproved model identity")
    model = build_local_recommender(directory, os.environ["ML_NER_MODEL_DIR"], device="cpu", top_k=3)
    if (model.top_k != 3 or model.ranker.model_version != RANKER
            or model.matcher.model_name != ENCODER
            or model.matcher.embedder.revision != REVISION
            or not all(getattr(part, "available", False) for part in (model.ner, model.matcher, model.ranker))
            or not isinstance(model.matcher._job_vectors, dict)):
        raise ValueError("loaded model mismatch")
    return model


class Server(ThreadingHTTPServer):
    daemon_threads = True
    allow_reuse_address = True

    def __init__(self, address, state):
        self.state = state
        super().__init__(address, Handler)

    def get_request(self):
        connection, address = super().get_request()
        connection.settimeout(30)
        return connection, address

    def handle_error(self, request, client_address):
        print("http_failure", file=sys.stderr)


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.0"
    server_version = "MLService"
    sys_version = ""

    def log_message(self, *args):
        pass

    def send_error(self, code, message=None, explain=None):
        self.error(code, "bad_request", "Invalid HTTP request")

    def reply(self, status, value):
        body = json.dumps(value, allow_nan=False, ensure_ascii=True).encode()
        self.close_connection = True
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.send_header("Connection", "close")
        self.end_headers()
        self.wfile.write(body)

    def error(self, status, code, message):
        self.reply(status, {"error": {"code": code, "message": message}})

    def do_GET(self):
        if self.path != "/readyz":
            return self.error(400, "bad_request", "Unknown endpoint")
        if self.server.state.model is None:
            return self.error(503, "not_ready", "Model unavailable")
        self.reply(200, {"ready": True})

    def do_POST(self):
        if self.path != "/recommend":
            return self.error(400, "bad_request", "Unknown endpoint")
        state = self.server.state
        auth = self.headers.get_all("Authorization", [])
        expected = b"Bearer " + state.token
        if len(auth) != 1 or not hmac.compare_digest(auth[0].encode("utf-8"), expected):
            return self.error(401, "unauthorized", "Invalid service token")
        if state.model is None:
            return self.error(503, "not_ready", "Model unavailable")
        lengths = self.headers.get_all("Content-Length", [])
        if self.headers.get("Transfer-Encoding") or len(lengths) != 1 or not lengths[0].isascii() or not lengths[0].isdigit():
            return self.error(400, "bad_request", "Invalid request framing")
        # Compare length text first: never parse an unbounded integer.
        length_text = lengths[0].lstrip("0") or "0"
        if len(length_text) > 7 or int(length_text) > BODY_CAP:
            return self.error(413, "payload_too_large", "Body exceeds 1 MiB")
        if self.headers.get_content_type() != "application/json":
            return self.error(400, "bad_request", "Expected application/json")
        if not state.worker.acquire(blocking=False):
            return self.error(429, "busy", "Inference busy")
        try:
            try:
                length = int(length_text)
                raw = self.rfile.read(length)
                if len(raw) != length:
                    raise ValueError("truncated body")
                candidate, jobs = parse_payload(raw)
            except (ValueError, TypeError, KeyError, RecursionError, socket.timeout):
                return self.error(400, "bad_request", "Invalid recommendation request")
            try:
                result = state.recommend(candidate, jobs)
                self.reply(200, result)
            except Exception:
                print("inference_failed", file=sys.stderr)
                self.error(500, "inference_failed", "Inference failed")
        finally:
            state.worker.release()


def main():
    # Dependency warnings may include user text; only fixed service events are logged.
    logging.disable(logging.CRITICAL)
    state = State(os.environ.get("ML_SERVICE_TOKEN", ""))
    try:
        state.model = load_model()
        state.model.ner = CheckedNer(state.model.ner)
        print("model_loaded", file=sys.stderr)
    except Exception:
        print("model_load_failed", file=sys.stderr)
    server = Server((os.environ.get("ML_HOST", "127.0.0.1"), int(os.environ.get("ML_PORT", "8090"))), state)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.server_close()


if __name__ == "__main__":
    main()
