
import http.client
import json
import threading
import unittest
from types import SimpleNamespace
from unittest.mock import Mock

import service
from pipeline.schemas import RecommendationResponse, NerOutput


def payload():
    return {"candidate": {"student_id": "test", "narrative": "", "cv_text": None,
                          "manual_profile": None, "consent_at": None}, "jobs": []}


class HTTPTests(unittest.TestCase):
    def setUp(self):
        self.model = SimpleNamespace(ner=SimpleNamespace(extract=Mock(return_value=NerOutput(status="ok"))),
                                     matcher=SimpleNamespace(_job_vectors={}), recommend=Mock())
        def recommend(candidate, jobs):
            if candidate.narrative_valid:
                self.state.model.ner.extract(candidate.narrative, candidate.cv_text)
            return RecommendationResponse(student_id=candidate.student_id, generated_at="2026-11-26T00:00:00Z",
                model_version="ner:test;encoder:test;ranker:test", status="ok" if candidate.narrative_valid else "profil_tidak_lengkap")
        self.model.recommend.side_effect = recommend
        self.state = service.State("test-token", self.model)
        self.server = service.Server(("127.0.0.1", 0), self.state)
        self.thread = threading.Thread(target=self.server.serve_forever)
        self.thread.start()

    def tearDown(self):
        self.server.shutdown()
        self.server.server_close()
        self.thread.join()

    def request(self, data=None, token="test-token", method="POST", path="/recommend", raw=None, extra=None):
        connection = http.client.HTTPConnection(*self.server.server_address, timeout=5)
        headers = {"Authorization": "Bearer " + token, "Content-Type": "application/json"}
        headers.update(extra or {})
        connection.request(method, path, raw if raw is not None else json.dumps(data), headers)
        response = connection.getresponse()
        result = response.status, json.loads(response.read())
        connection.close()
        return result

    def test_incomplete_and_empty_success(self):
        status, result = self.request(payload())
        self.assertEqual((status, result["status"], result["items"]), (200, "profil_tidak_lengkap", []))
        self.model.ner.inner.extract.assert_not_called()
        data = payload()
        data["candidate"].update(narrative="Python backend development and SQL project experience. " * 2,
                                  consent_at="2026-11-26T00:00:00Z")
        status, result = self.request(data)
        self.assertEqual((status, result["status"], result["top_k"], result["items"]), (200, "ok", 3, []))

    def test_readiness_and_notready(self):
        self.assertEqual(self.request(method="GET", path="/readyz")[0], 200)
        self.state.model = None
        self.assertEqual(self.request(method="GET", path="/readyz")[0], 503)
        self.assertEqual(self.request(payload())[0], 503)

    def test_token(self):
        self.assertEqual(self.request(payload(), token="wrong")[0], 401)
        with self.assertRaises(ValueError):
            service.State("")

    def test_oversized(self):
        self.assertEqual(self.request(raw=b"", extra={"Content-Length": str(service.BODY_CAP + 1)})[0], 413)

    def test_busy(self):
        self.state.worker.acquire()
        try:
            self.assertEqual(self.request(payload())[0], 429)
            self.assertEqual(self.request(method="GET", path="/readyz")[0], 200)
        finally:
            self.state.worker.release()

    def test_validation(self):
        variants = []
        for change in ({"extra": 1}, {"narrative": "x" * 2001}, {"cv_text": "x" * 100001},
                       {"manual_profile": {"bad": []}}, {"student_id": 3}, {"narrative": "x" * 50}):
            data = payload()
            data["candidate"].update(change)
            variants.append(data)
        data = payload()
        data["jobs"] = [{}] * 201
        variants.append(data)
        variants += [{"candidate": {}, "jobs": []}, {**payload(), "top_k": 5}]
        for data in variants:
            with self.subTest(data=str(data)[:40]):
                self.assertEqual(self.request(data)[0], 400)
        self.assertEqual(self.request(raw='{"candidate":{},"candidate":{},"jobs":[]}')[0], 400)
        self.assertEqual(self.request(raw='{"candidate":NaN,"jobs":[]}')[0], 400)

    def test_mitra_metadata_and_external_url_validation(self):
        job = dict(job_id="internal-original", title="Go Intern", category="Backend Developer Intern",
                   description="Original partner description", required_skills=["Go", "SQL"],
                   optional_skills=["Docker"], source="mitra", source_url="",
                   collected_at="2026-11-26T00:00:00Z")
        data = payload()
        data["jobs"] = [job]
        self.assertEqual(self.request(data, token="wrong")[0], 401)
        self.model.recommend.assert_not_called()
        self.assertEqual(self.request(data)[0], 200)
        forwarded = self.model.recommend.call_args.args[1][0].to_dict()
        for key, value in job.items():
            self.assertEqual(forwarded[key], value)
        for source in ("Glints", "Dealls", "Jobstreet", "LinkedIn"):
            with self.subTest(source=source):
                self.model.recommend.reset_mock()
                job["source"] = source
                self.assertEqual(self.request(data)[0], 400)
                self.model.recommend.assert_not_called()

    def test_stale_cache(self):
        job = dict(job_id="j1", title="Python Intern", category="Backend Developer Intern",
                   source="LinkedIn", source_url="https://example.test/j1", collected_at="2026-11-26")
        data = payload()
        data["jobs"] = [job]
        self.request(data)
        cache = self.model.matcher._job_vectors
        cache["j1"] = [1]
        self.request(data)
        self.assertEqual(cache, {"j1": [1]})
        job["description"] = "Updated catalog content"
        self.request(data)
        self.assertEqual(cache, {})
        cache["j1"] = [2]
        self.request(payload())
        self.assertEqual(cache, {})

    def test_swallowed_ner_failure(self):
        self.model.ner.inner.extract.side_effect = RuntimeError("private text")
        def swallowed(candidate, jobs):
            try:
                self.model.ner.extract(candidate.narrative, None)
            except Exception:
                pass
            return RecommendationResponse(student_id="test", generated_at="now",
                                          model_version="ner:test;encoder:test;ranker:test")
        self.model.recommend.side_effect = swallowed
        self.assertEqual(self.request(payload()), (500, {"error": {"code": "inference_failed", "message": "Inference failed"}}))


if __name__ == "__main__":
    unittest.main()
