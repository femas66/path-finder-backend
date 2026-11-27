"""Real offline HTTP smoke. Uses configured trusted local model artifacts."""
import http.client
import json
import logging
import threading

import service


def main():
    logging.disable(logging.CRITICAL)
    model = service.load_model()
    state = service.State("local-smoke-only", model)
    server = service.Server(("127.0.0.1", 0), state)
    thread = threading.Thread(target=server.serve_forever)
    thread.start()
    def request(method, path, data=None):
        connection = http.client.HTTPConnection(*server.server_address, timeout=30)
        connection.request(method, path, json.dumps(data) if data is not None else None,
                           {"Authorization": "Bearer local-smoke-only", "Content-Type": "application/json"})
        response = connection.getresponse()
        result = response.status, json.loads(response.read())
        connection.close()
        return result
    try:
        assert request("GET", "/readyz") == (200, {"ready": True})
        data = {"candidate": {"student_id": "service-smoke", "narrative":
                "Saya menguasai Python, SQL, dan Git serta mampu bekerja sama dalam tim untuk membangun aplikasi backend.",
                "consent_at": "2026-11-26T00:00:00Z"}, "jobs": [
            {"job_id": f"smoke-{i}", "title": "Python Intern", "category": "Backend Developer Intern",
             "required_skills": skills, "source": "LinkedIn", "source_url": f"https://example.test/{i}",
             "collected_at": "2026-11-26T00:00:00Z"}
            for i, skills in enumerate((["Python", "SQL"], ["Java"], ["Git"], ["React"]))]}
        status, result = request("POST", "/recommend", data)
        assert status == 200, (status, result)
        assert result["status"] == "ok" and result["top_k"] == 3
        assert [item["rank"] for item in result["items"]] == [1, 2, 3]
        assert f"ranker:{service.RANKER}" in result["model_version"]
        for item in result["items"]:
            assert type(item["matching_score"]) is int and 0 <= item["matching_score"] <= 100
            assert type(item["readiness_score"]) is int and 0 <= item["readiness_score"] <= 100
        data["jobs"][0]["description"] = "Updated Python SQL backend role"
        assert request("POST", "/recommend", data)[0] == 200
        data["jobs"] = []
        assert request("POST", "/recommend", data)[1]["items"] == []
        data["candidate"] = {"student_id": "service-smoke", "narrative": ""}
        assert request("POST", "/recommend", data)[1]["status"] == "profil_tidak_lengkap"
        print(json.dumps({"check": "passed", "response": result}))
    finally:
        server.shutdown()
        server.server_close()
        thread.join()


if __name__ == "__main__":
    main()
