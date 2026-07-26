#!/usr/bin/env python3
import os
import json
import base64
import requests
import sys

GITHUB_TOKEN = "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"
OWNER = "zhangzhenling"
REPO = "TCM-History-AI"
BRANCH = "trae/agent-aoVLMZ"
API_BASE = "https://api.github.com"

headers = {
    "Authorization": f"token {GITHUB_TOKEN}",
    "Accept": "application/vnd.github.v3+json"
}

def get_latest_commit():
    url = f"{API_BASE}/repos/{OWNER}/{REPO}/git/ref/heads/{BRANCH}"
    resp = requests.get(url, headers=headers)
    resp.raise_for_status()
    return resp.json()["object"]["sha"]

def get_commit_tree(commit_sha):
    url = f"{API_BASE}/repos/{OWNER}/{REPO}/git/commits/{commit_sha}"
    resp = requests.get(url, headers=headers)
    resp.raise_for_status()
    return resp.json()["tree"]["sha"]

def create_blob(content):
    url = f"{API_BASE}/repos/{OWNER}/{REPO}/git/blobs"
    data = {
        "content": content,
        "encoding": "utf-8"
    }
    resp = requests.post(url, headers=headers, json=data)
    resp.raise_for_status()
    return resp.json()["sha"]

def create_tree(base_tree_sha, files):
    url = f"{API_BASE}/repos/{OWNER}/{REPO}/git/trees"
    tree = []
    for f in files:
        tree.append({
            "path": f["path"],
            "mode": "100644",
            "type": "blob",
            "sha": f["sha"]
        })
    data = {
        "base_tree": base_tree_sha,
        "tree": tree
    }
    resp = requests.post(url, headers=headers, json=data)
    resp.raise_for_status()
    return resp.json()["sha"]

def create_commit(tree_sha, parent_sha, message):
    url = f"{API_BASE}/repos/{OWNER}/{REPO}/git/commits"
    data = {
        "message": message,
        "tree": tree_sha,
        "parents": [parent_sha]
    }
    resp = requests.post(url, headers=headers, json=data)
    resp.raise_for_status()
    return resp.json()["sha"]

def update_ref(commit_sha):
    url = f"{API_BASE}/repos/{OWNER}/{REPO}/git/refs/heads/{BRANCH}"
    data = {
        "sha": commit_sha
    }
    resp = requests.patch(url, headers=headers, json=data)
    resp.raise_for_status()
    return resp.json()

def push_files(files, message):
    print(f"Pushing {len(files)} files...")
    
    latest_commit = get_latest_commit()
    print(f"  Latest commit: {latest_commit}")
    
    base_tree = get_commit_tree(latest_commit)
    print(f"  Base tree: {base_tree}")
    
    blobs = []
    for i, f in enumerate(files):
        print(f"  Creating blob {i+1}/{len(files)}: {f['path']}")
        sha = create_blob(f["content"])
        blobs.append({
            "path": f["path"],
            "sha": sha
        })
    
    new_tree = create_tree(base_tree, blobs)
    print(f"  New tree: {new_tree}")
    
    new_commit = create_commit(new_tree, latest_commit, message)
    print(f"  New commit: {new_commit}")
    
    update_ref(new_commit)
    print(f"  Updated ref: {BRANCH}")
    
    return new_commit

def read_file_content(filepath, workspace):
    full_path = os.path.join(workspace, filepath)
    try:
        if os.path.getsize(full_path) == 0:
            return ""
        with open(full_path, 'r', encoding='utf-8') as f:
            return f.read()
    except UnicodeDecodeError:
        with open(full_path, 'rb') as f:
            return base64.b64encode(f.read()).decode('ascii')
    except Exception as e:
        print(f"Error reading {filepath}: {e}", file=sys.stderr)
        return ""

def get_files_from_git_diff(workspace):
    import subprocess
    result = subprocess.run(
        ["git", "-c", "core.quotepath=false", "diff", "origin/main", "--name-only"],
        cwd=workspace,
        capture_output=True,
        text=True
    )
    files = [f.strip() for f in result.stdout.strip().split('\n') if f.strip()]
    return files

def main():
    workspace = "/workspace"
    commit_message = "feat: RAG 写入侧流水线 + Prompt 激活管理 + Helm/K8s 部署清单 + Flutter 移动端骨架"
    
    files = get_files_from_git_diff(workspace)
    print(f"Total files to push: {len(files)}")
    
    batch_size = 25
    batches = [files[i:i+batch_size] for i in range(0, len(files), batch_size)]
    print(f"Total batches: {len(batches)}")
    
    for i, batch in enumerate(batches):
        print(f"\n=== Batch {i+1}/{len(batches)} ===")
        batch_files = []
        for filepath in batch:
            content = read_file_content(filepath, workspace)
            batch_files.append({
                "path": filepath,
                "content": content
            })
        
        batch_message = f"{commit_message} (batch {i+1}/{len(batches)})"
        push_files(batch_files, batch_message)
        print(f"Batch {i+1} complete!")
    
    print("\n=== All batches complete! ===")

if __name__ == "__main__":
    main()
