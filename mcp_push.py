#!/usr/bin/env python3
import subprocess
import json
import sys
import os
import base64

WORKSPACE = "/workspace"
COMMIT_MESSAGE = "feat: RAG 写入侧流水线 + Prompt 激活管理 + Helm/K8s 部署清单 + Flutter 移动端骨架"
OWNER = "zhangzhenling"
REPO = "TCM-History-AI"
BRANCH = "trae/agent-aoVLMZ"
BATCH_SIZE = 25

class MCPClient:
    def __init__(self, command, env=None):
        self.process = subprocess.Popen(
            command,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            env=env,
            text=True,
            bufsize=1
        )
        self.request_id = 0
    
    def send_request(self, method, params=None):
        self.request_id += 1
        request = {
            "jsonrpc": "2.0",
            "id": self.request_id,
            "method": method,
            "params": params or {}
        }
        request_str = json.dumps(request) + "\n"
        self.process.stdin.write(request_str)
        self.process.stdin.flush()
        
        while True:
            line = self.process.stdout.readline()
            if not line:
                break
            try:
                response = json.loads(line.strip())
                if "id" in response and response["id"] == self.request_id:
                    if "error" in response:
                        raise Exception(f"MCP Error: {response['error']}")
                    return response.get("result")
            except json.JSONDecodeError:
                continue
        return None
    
    def initialize(self):
        return self.send_request("initialize", {
            "protocolVersion": "2024-11-05",
            "capabilities": {},
            "clientInfo": {
                "name": "python-client",
                "version": "1.0.0"
            }
        })
    
    def list_tools(self):
        return self.send_request("tools/list")
    
    def call_tool(self, name, arguments):
        return self.send_request("tools/call", {
            "name": name,
            "arguments": arguments
        })

def get_files_from_git_diff(workspace):
    result = subprocess.run(
        ["git", "-c", "core.quotepath=false", "diff", "origin/main", "--name-only"],
        cwd=workspace,
        capture_output=True,
        text=True
    )
    files = [f.strip() for f in result.stdout.strip().split('\n') if f.strip()]
    return files

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

def main():
    workspace = WORKSPACE
    
    files = get_files_from_git_diff(workspace)
    print(f"Total files to push: {len(files)}")
    
    batches = [files[i:i+BATCH_SIZE] for i in range(0, len(files), BATCH_SIZE)]
    print(f"Total batches: {len(batches)}")
    
    env = os.environ.copy()
    env["GITHUB_PERSONAL_ACCESS_TOKEN"] = "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"
    
    client = MCPClient(
        ["npx", "-y", "@modelcontextprotocol/server-github"],
        env=env
    )
    
    print("Initializing MCP client...")
    init_result = client.initialize()
    print(f"Initialized: {init_result is not None}")
    
    print("\nListing tools...")
    tools = client.list_tools()
    tool_names = [t["name"] for t in tools.get("tools", [])]
    print(f"Available tools: {tool_names}")
    
    if "push_files" not in tool_names:
        print("ERROR: push_files tool not found!")
        return
    
    for i, batch in enumerate(batches):
        print(f"\n=== Batch {i+1}/{len(batches)} ===")
        batch_files = []
        for filepath in batch:
            content = read_file_content(filepath, workspace)
            batch_files.append({
                "path": filepath,
                "content": content
            })
            print(f"  Prepared: {filepath} ({len(content)} chars)")
        
        batch_message = f"{COMMIT_MESSAGE} (batch {i+1}/{len(batches)})"
        
        print(f"  Calling push_files with {len(batch_files)} files...")
        result = client.call_tool("push_files", {
            "owner": OWNER,
            "repo": REPO,
            "branch": BRANCH,
            "files": batch_files,
            "message": batch_message
        })
        print(f"  Result: {json.dumps(result, indent=2, ensure_ascii=False)}")
        print(f"Batch {i+1} complete!")
    
    print("\n=== All batches complete! ===")

if __name__ == "__main__":
    main()
