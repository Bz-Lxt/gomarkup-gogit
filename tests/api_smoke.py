import os
import time
import uuid

import requests

BASE = os.environ.get("GOGIT_BASE_URL", "http://127.0.0.1:41783").rstrip("/")


def api(method, path, **kw):
    r = requests.request(method, BASE + path, timeout=20, **kw)
    return r


def test_health_and_spa_shell():
    r = api("GET", "/api/v1/health")
    assert r.status_code == 200, r.text
    body = r.json()["data"]
    assert body["status"] == "ok"
    assert "time" in body
    page = api("GET", "/")
    assert page.status_code == 200
    html = page.text
    assert "<div id=\"app\">" in html
    assert "main.js" in html or "/assets/" in html


def test_repo_info():
    r = api("GET", "/api/v1/repo")
    assert r.status_code == 200, r.text
    data = r.json()["data"]
    assert data["hash_algo"] in ("sha1", "sha256")
    assert data["current_branch"]


def test_add_commit_branch_merge_object():
    stamp = time.strftime("%Y%m%d%H%M%S") + "-" + uuid.uuid4().hex[:6]
    path = f"smoke/{stamp}.txt"
    content = f"smoke-{stamp}\n"
    r = api("PUT", "/api/v1/files", json={"path": path, "content": content})
    assert r.status_code == 200, r.text

    r = api("POST", "/api/v1/index/add", json={"paths": [path]})
    assert r.status_code == 200, r.text

    r = api("GET", "/api/v1/status")
    assert r.status_code == 200
    staged = [s["path"] for s in r.json()["data"]["staged"]]
    assert path in staged

    r = api("POST", "/api/v1/commits", json={"message": f"smoke {stamp}", "author": "QA <qa@gogit.local>"})
    assert r.status_code == 201, r.text
    commit = r.json()["data"]
    assert commit["hash"]
    assert commit["committed_at"]

    branch = f"smoke/{stamp}"
    r = api("POST", "/api/v1/branches", json={"name": branch})
    assert r.status_code == 201, r.text

    r = api("POST", "/api/v1/checkout", json={"name": branch})
    assert r.status_code == 200, r.text

    r = api("PUT", "/api/v1/files", json={"path": path, "content": content + "side\n"})
    assert r.status_code == 200
    assert api("POST", "/api/v1/index/add", json={"paths": [path]}).status_code == 200
    r = api("POST", "/api/v1/commits", json={"message": f"side {stamp}", "author": "QA <qa@gogit.local>"})
    assert r.status_code == 201, r.text

    # back to main then merge
    repo = api("GET", "/api/v1/repo").json()["data"]
    main = "main"
    r = api("POST", "/api/v1/checkout", json={"name": main})
    assert r.status_code == 200, r.text

    r = api("POST", "/api/v1/merge", json={"branch": branch})
    assert r.status_code == 200, r.text
    merged = r.json()["data"]
    assert merged.get("commit") or merged.get("fast_forward") is not None

    obj = api("GET", f"/api/v1/objects/{commit['hash']}")
    assert obj.status_code == 200, obj.text
    assert obj.json()["data"]["type"] == "commit"

    r = api("GET", "/api/v1/files/content", params={"path": path})
    assert r.status_code == 200
    assert "side" in r.json()["data"]["content"]


def test_revparse_fsck_diff_unstage():
    stamp = time.strftime("%Y%m%d%H%M%S") + "-" + uuid.uuid4().hex[:6]
    path = f"smoke/extra-{stamp}.txt"
    assert api("PUT", "/api/v1/files", json={"path": path, "content": "v1\n"}).status_code == 200
    assert api("POST", "/api/v1/index/add", json={"paths": [path]}).status_code == 200
    c = api("POST", "/api/v1/commits", json={"message": f"extra {stamp}", "author": "QA <qa@gogit.local>"})
    assert c.status_code == 201, c.text
    assert api("PUT", "/api/v1/files", json={"path": path, "content": "v2\n"}).status_code == 200
    d = api("GET", "/api/v1/diff", params={"path": path, "side": "unstaged"})
    assert d.status_code == 200, d.text
    assert "v2" in d.json()["data"]["patch"]

    assert api("POST", "/api/v1/index/add", json={"paths": [path]}).status_code == 200
    u = api("POST", "/api/v1/index/unstage", json={"paths": [path]})
    assert u.status_code == 200, u.text
    staged = [s["path"] for s in u.json()["data"]["staged"]]
    assert path not in staged

    rp = api("GET", "/api/v1/rev-parse", params={"q": "HEAD"})
    assert rp.status_code == 200
    assert rp.json()["data"]["oid"]

    fs = api("GET", "/api/v1/fsck")
    assert fs.status_code == 200, fs.text
    assert fs.json()["data"]["ok"] is True


def test_validation_and_unknown():
    r = api("POST", "/api/v1/commits", json={"message": ""})
    assert r.status_code in (400, 422)
    r = api("GET", "/api/v1/does-not-exist")
    assert r.status_code == 404
    r = api("GET", "/api/v1/files/content?path=../etc/passwd")
    assert r.status_code == 400
