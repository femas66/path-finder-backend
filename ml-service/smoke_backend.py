"""Run real Go HTTP -> Python ML -> isolated PostgreSQL smoke; no downloads.

Run from any directory using sibling pathfinder-ml/.venv/bin/python.
Only synthetic fixture rows; PostgreSQL cluster and service live temporarily.
"""
import contextlib
import json
import logging
import os
from pathlib import Path
import secrets
import shutil
import socket
import subprocess
import tempfile
import threading

import service


def main():
    root = Path(__file__).resolve().parents[1]
    ml = root.parent / "pathfinder-ml"
    os.environ.setdefault("ML_TRAINING_DIR", str(ml / "models/ranking/reviewed-development-v1"))
    os.environ.setdefault("ML_NER_MODEL_DIR", str(ml / "models/ner/wandi-inference-v3/final"))
    os.environ["ML_THREADS"] = "1"
    logging.disable(logging.CRITICAL)
    # Factory loads actual pinned local artifacts; failure aborts, never mocks.
    model = service.load_model()
    token = secrets.token_urlsafe(32)
    server = service.Server(("127.0.0.1", 0), service.State(token, model))
    thread = threading.Thread(target=server.serve_forever)
    initdb = shutil.which("initdb")
    if not initdb:
        raise RuntimeError("initdb required on PATH")
    pg_ctl = str(Path(initdb).with_name("pg_ctl"))
    with tempfile.TemporaryDirectory(prefix="pathfinder-real-ml-") as temp:
        data = str(Path(temp) / "pg")
        log = str(Path(temp) / "postgres.log")
        # Loopback only; trust applies exclusively to this disposable cluster.
        subprocess.run([initdb, "-D", data, "-U", "smoke", "-A", "trust", "--no-locale"], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE)
        with contextlib.closing(socket.socket()) as sock:
            sock.bind(("127.0.0.1", 0))
            port = sock.getsockname()[1]
        started = False
        thread.start()
        try:
            subprocess.run([pg_ctl, "-D", data, "-l", log, "-o", f"-h 127.0.0.1 -p {port} -k ''", "-w", "start"], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE)
            started = True
            env = os.environ.copy()
            env.update(PATHFINDER_REAL_ML_URL=f"http://127.0.0.1:{server.server_port}",
                       PATHFINDER_REAL_ML_TOKEN=token,
                       PATHFINDER_JOB_TEST_DSN=f"postgres://smoke@127.0.0.1:{port}/postgres?sslmode=disable",
                       OMP_NUM_THREADS="1", MKL_NUM_THREADS="1", OPENBLAS_NUM_THREADS="1")
            subprocess.run(["go", "test", "./test", "-run", "^TestRecommendationIntegration", "-count=1", "-v", "-timeout", "120s"], cwd=root, env=env, check=True, timeout=150)
            subprocess.run(["go", "test", "./test", "-run", "^TestRealMLRecommendation$",  "-count=1", "-v", "-timeout", "180s"], cwd=root, env=env, check=True, timeout=210)
            print(json.dumps({"check": "passed", "boundary": "authenticated mitra job creation -> Go HTTP -> Python -> local NER/E5/GB -> PostgreSQL catalog", "fixtures": "synthetic, disposable"}))
        finally:
            server.shutdown()
            server.server_close()
            thread.join()
            if started:
                subprocess.run([pg_ctl, "-D", data, "-m", "immediate", "-w", "stop"], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE)


if __name__ == "__main__":
    main()
