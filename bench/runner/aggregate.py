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

    # data_map[case][gateway] = json_obj
    data_map = {}
    all_gateways = set()
    all_cases = set()

    for f in files:
        try:
            with open(f, 'r') as fd:
                entry = json.load(fd)
                c = entry.get("case")
                g = entry.get("gateway")
                if c and g:
                    if c not in data_map:
                        data_map[c] = {}
                    data_map[c][g] = entry
                    all_gateways.add(g)
                    all_cases.add(c)
        except Exception as e:
            print(f"Error reading {f}: {e}", file=sys.stderr)

    sorted_gateways = sorted(list(all_gateways))
    if "homebrew" in sorted_gateways: sorted_gateways.insert(0, sorted_gateways.pop(sorted_gateways.index("homebrew")))

    sorted_cases = sorted(list(all_cases))

    headers = ["Case", "Metric"] + sorted_gateways
    print("| " + " | ".join(headers) + " |")
    print("| " + " | ".join(["---"] * len(headers)) + " |")

    # 5. 输出数据行
    metrics = [
        ("RPS", "rps"),
        ("P50 (ms)", "p50"),
        ("P90 (ms)", "p90"),
        ("P99 (ms)", "p99")
    ]

    for case in sorted_cases:
        first_row = True
        for label, key in metrics:
            row = []

            if first_row:
                row.append(f"**{case}**")
                first_row = False
            else:
                row.append("")

            row.append(label)

            for g in sorted_gateways:
                entry = data_map.get(case, {}).get(g)
                if not entry:
                    row.append("-")
                    continue

                val = ""
                if key == "rps":
                    val = f"{entry.get('rps', 0):.2f}"
                else:
                    lat = entry.get('latency', {})
                    val = f"{lat.get(key, 0):.2f}"

                row.append(val)

            print("| " + " | ".join(row) + " |")

if __name__ == "__main__":
    main()
