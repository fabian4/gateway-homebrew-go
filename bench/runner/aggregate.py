import sys
import os
import json
import glob

def main():
    if len(sys.argv) < 2:
        print("Usage: python aggregate.py <results_dir>")
        sys.exit(1)

    results_dir = sys.argv[1]
    files = glob.glob(os.path.join(results_dir, "*.json"))
    
    data = []
    for f in files:
        try:
            with open(f, 'r') as fd:
                data.append(json.load(fd))
        except Exception as e:
            print(f"Error reading {f}: {e}", file=sys.stderr)

    # Group by case
    cases = {}
    gateways = set()
    for d in data:
        c = d.get("case")
        g = d.get("gateway")
        if not c or not g:
            continue
        if c not in cases:
            cases[c] = {}
        cases[c][g] = d
        gateways.add(g)

    sorted_gateways = sorted(list(gateways))
    sorted_cases = sorted(list(cases.keys()))

    print("# Benchmark Results")
    print("")
    
    for c in sorted_cases:
        print(f"## Case: {c}")
        print("| Gateway | RPS | P50 (ms) | P90 (ms) | P99 (ms) |")
        print("|---|---|---|---|---|")
        
        for g in sorted_gateways:
            res = cases[c].get(g)
            if not res:
                print(f"| {g} | N/A | N/A | N/A | N/A |")
                continue
            
            rps = f"{res.get('rps', 0):.2f}"
            lat = res.get('latency', {})
            p50 = f"{lat.get('p50', 0):.2f}"
            p90 = f"{lat.get('p90', 0):.2f}"
            p99 = f"{lat.get('p99', 0):.2f}"
            
            print(f"| {g} | {rps} | {p50} | {p90} | {p99} |")
        print("")

if __name__ == "__main__":
    main()
