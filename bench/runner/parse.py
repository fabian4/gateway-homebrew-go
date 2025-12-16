import sys
import re
import json
import os

def parse_time(value):
    if value.endswith('ms'):
        return float(value[:-2])
    if value.endswith('s'):
        return float(value[:-1]) * 1000
    if value.endswith('us'):
        return float(value[:-2]) / 1000
    return 0.0

def parse_wrk_output(content, gateway, case):
    result = {
        "gateway": gateway,
        "case": case,
        "rps": 0.0,
        "latency": {}
    }

    # Requests/sec:   4730.27
    rps_match = re.search(r'Requests/sec:\s+(\d+\.?\d*)', content)
    if rps_match:
        result["rps"] = float(rps_match.group(1))

    # Latency Distribution
    #  50%    1.10ms
    #  90%    2.00ms
    #  99%    5.00ms
    latency_map = {
        "50%": "p50",
        "90%": "p90",
        "99%": "p99"
    }
    
    for line in content.split('\n'):
        line = line.strip()
        for k, v in latency_map.items():
            if line.startswith(k):
                parts = line.split()
                if len(parts) >= 2:
                    result["latency"][v] = parse_time(parts[1])

    return result

if __name__ == "__main__":
    if len(sys.argv) < 4:
        print("Usage: python parse.py <file> <gateway> <case>")
        sys.exit(1)

    filepath = sys.argv[1]
    gateway = sys.argv[2]
    case = sys.argv[3]

    try:
        with open(filepath, 'r') as f:
            content = f.read()
        
        data = parse_wrk_output(content, gateway, case)
        
        # Output JSON to stdout or a file? 
        # Let's write to a .json file next to the .txt file
        json_path = filepath.replace('.txt', '.json')
        with open(json_path, 'w') as f:
            json.dump(data, f, indent=2)
            
        print(json.dumps(data, indent=2))
        
    except Exception as e:
        print(f"Error parsing results: {e}", file=sys.stderr)
        sys.exit(1)
