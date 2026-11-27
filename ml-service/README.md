# Internal ML HTTP service

Stdlib HTTP boundary; ML dependencies come from the existing sibling environment.
No pip installs, training, downloads, Go changes, or model artifacts added.

## Launch

From `pathfinder-be/ml-service/`:

```sh
export ML_SERVICE_TOKEN='replace-with-a-long-random-internal-secret'
export ML_TRAINING_DIR='../../pathfinder-ml/models/ranking/reviewed-development-v1'
export ML_NER_MODEL_DIR='../../pathfinder-ml/models/ner/wandi-inference-v3/final'
../../pathfinder-ml/.venv/bin/python service.py
```

Required: `ML_SERVICE_TOKEN`, `ML_TRAINING_DIR`, `ML_NER_MODEL_DIR`.
Optional: `ML_PIPELINE_SRC` (default sibling `pathfinder-ml/src`), `ML_HOST`
(default `127.0.0.1`), `ML_PORT` (8090), `ML_THREADS` (1; range 1..64 CPU threads).
Paths resolve relative to working directory; absolute paths recommended in deployment.
`ML_THREADS` does not increase inference concurrency.

**Security:** internal HTTP only, not a public production web server. Keep loopback
or a private isolated network; use a TLS proxy across hosts. Token must be nonempty
ASCII without surrounding whitespace. Never expose or commit it. Pipeline source,
training reports, joblib bundles, NER artifacts must be operator-controlled trusted
local files: joblib can execute code; hashes do not establish authenticity.

Offline flags `HF_HUB_OFFLINE=1`, `TRANSFORMERS_OFFLINE=1` forced before imports.
Model loaded once before serving. Factory checks provenance/configuration, bundle
hashes, manifest, encoder identity, local-only loaders. Service additionally pins
encoder `intfloat/multilingual-e5-small`, revision
`614241f622f53c4eeff9890bdc4f31cfecc418b3`, selected GB ranker
`gb-e5-reviewed-dev-v1` (actual current report selection, not historical RF).
Changing these requires deliberate code review. No fallback or runtime reload.
Missing/invalid token exits startup; model/config/dependency load failure logs only
`model_load_failed`, then serves not-ready. Successful startup logs `model_loaded`.
Recovery requires fixing artifacts/config and restarting.

## HTTP contract

`GET /readyz`: unauthenticated, 200 `{"ready":true}` after load;
503 `{"error":{"code":"not_ready","message":"Model unavailable"}}` otherwise.
Readiness means loaded, not idle; remains 200 during inference.

`POST /recommend`: `Authorization: Bearer <ML_SERVICE_TOKEN>`,
`Content-Type: application/json`, one valid `Content-Length`; no chunked bodies.
Success 200 is **raw ML RecommendationResponse**, not wrapped in `data`:
`student_id`, `generated_at`, `schema_version`, `model_version`, `top_k`, `status`,
`items`. `top_k=3` fixed. Short narrative is valid input: 200 with
`status=profil_tidak_lengkap`, `items=[]`. Complete candidate + empty catalog:
200 `status=ok`, `items=[]`. Schema defaults retained; no invented profile/skills.

Example contract-shaped request (synthetic HTTP example, not production data):

```json
{
  "candidate": {
    "student_id": "example-student",
    "narrative": "Saya menguasai Python, SQL, dan Git untuk membangun aplikasi backend bersama tim.",
    "cv_text": null,
    "manual_profile": {"hard_skills": ["Python"], "soft_skills": [], "interests": [], "experience": []},
    "consent_at": "2026-11-26T00:00:00Z"
  },
  "jobs": [{
    "job_id": "example-job",
    "title": "Backend Intern",
    "category": "Backend Developer Intern",
    "description": "Membangun aplikasi backend.",
    "required_skills": ["Python", "SQL"],
    "optional_skills": ["Git"],
    "requirements_text": "Python dan SQL",
    "source": "LinkedIn",
    "source_url": "https://example.test/jobs/example-job",
    "collected_at": "2026-11-26T00:00:00Z",
    "is_synthetic": true
  }]
}
```

Product catalog source is `mitra`, created through authenticated `POST /api/mitra/jobs`.
Only internal `mitra` jobs may have an empty `source_url`; external sources
(`Glints`, `Dealls`, `Jobstreet`, `LinkedIn`) require a nonblank URL. The backend
preserves stored ID, source, URL, category, description, and skill lists without
relabeling jobs as LinkedIn or inventing URLs. Creation retains its existing
normalization/escaping rules; recommendation forwards the persisted values.

Timestamp provenance: the jobs table exposes `created_at` and `updated_at`, not a
collection timestamp. The existing ML wire field `collected_at` carries the actual
persisted `jobs.created_at` in UTC RFC3339; for `mitra` this means internal record
creation, **not scraping/collection time**. No collection event or timestamp is
fabricated. For external records this mapping also does not prove collection time.
The schema is imported from `pipeline.schemas`; this service has no separate source
allowlist. Integration tests mock inference, not the creation/recommendation handlers;
HTTP service tests use a mock model, not trained-model validation.

Unknown fields, duplicate JSON keys, nonfinite constants, duplicate job IDs,
invalid schema types/enums rejected. Consent validation follows CandidateInput.
Limits: body 1 MiB bytes; narrative 2,000 characters; CV 100,000; jobs 200;
manual/skill lists 200 entries each; list-entry text 2,000; student ID 256;
consent 128; job description/requirements 20,000 each; other job strings 2,000.
Body cap bounds aggregate payload including all text. Excess field/list counts or
text lengths return 400; body byte cap returns 413. No truncation.

Errors always `{"error":{"code":"...","message":"..."}}`:

| HTTP | code | message |
|---|---|---|
| 400 | bad_request | Invalid recommendation request / Invalid request framing / Expected application/json / Unknown endpoint / Invalid HTTP request |
| 401 | unauthorized | Invalid service token |
| 413 | payload_too_large | Body exceeds 1 MiB |
| 429 | busy | Inference busy |
| 503 | not_ready | Model unavailable |
| 500 | inference_failed | Inference failed |

Token checked before readiness/body; no exception details or PII returned/logged.
Default request/dependency logging disabled. Fixed failure events only.
One nonblocking semaphore covers body parsing + inference + cache mutation;
concurrent recommendations return 429, no waiting inference queue. HTTP uses
ThreadingHTTPServer, so readiness is not blocked by inference. Socket timeout 30s
(read/write inactivity, not a cancellable model deadline). No forced thread kill.
HTTP connections close after each response. A sender streaming an oversized body
may see connection reset after early rejection; do not retry automatically.

Matcher has no public cache-clear method (`register_job_vectors` only updates).
Under the single worker, SHA-256 of the complete serialized catalog invalidates
`matcher._job_vectors` on any content/order change, including same-ID edits and
removed jobs. Model remains loaded. NER adapter records actual exceptions/non-ok
outputs so upstream swallowed failures become 500 instead of fabricated success.
Legitimate no-entity extraction remains valid. Existing ML score/status semantics
otherwise preserved.

## Checks

```sh
../../pathfinder-ml/.venv/bin/python -m unittest -v test_service.py
# Same training/NER environment as launch; ephemeral loopback HTTP server:
../../pathfinder-ml/.venv/bin/python smoke_service.py
# From pathfinder-be/: isolated PostgreSQL integration + real mitra catalog smoke
../pathfinder-ml/.venv/bin/python ml-service/smoke_backend.py
```

Smoke uses actual local NER/E5/GB, Top-3 ranks, integer scores, catalog edit,
empty catalog and incomplete profile. Functional verification only; not a
relevance benchmark. Backend smoke creates jobs through authenticated
`POST /api/mitra/jobs`, verifies server-assigned company/source and empty URL,
closes one through PATCH, then checks real Top-3 and null scores for a no-skill
profile. Runs recommendation integration tests against the same disposable
PostgreSQL cluster first. Requires `initdb` and `pg_ctl` on PATH.
Sibling source/contracts remain unchanged.
