#!/usr/bin/env python3
import subprocess
import json
import sys
import os

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
                stderr = self.process.stderr.read()
                raise Exception(f"Process exited. Stderr: {stderr}")
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

def main():
    env = os.environ.copy()
    env["GITHUB_PERSONAL_ACCESS_TOKEN"] = "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"
    
    client = MCPClient(
        ["npx", "-y", "@modelcontextprotocol/server-github"],
        env=env
    )
    
    print("Initializing MCP client...")
    init_result = client.initialize()
    print(f"Initialized: {init_result is not None}")
    
    print("\nTesting search_repositories...")
    try:
        result = client.call_tool("search_repositories", {
            "query": "TCM-History-AI"
        })
        print(f"Result: {json.dumps(result, indent=2, ensure_ascii=False)[:500]}")
    except Exception as e:
        print(f"Error: {e}")
    
    print("\nTesting list_commits...")
    try:
        result = client.call_tool("list_commits", {
            "owner": "zhangzhenling",
            "repo": "TCM-History-AI"
        })
        print(f"Result: {json.dumps(result, indent=2, ensure_ascii=False)[:500]}")
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    main()
