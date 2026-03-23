#!/usr/bin/env python3

import re
import sys
import os
from collections import defaultdict


BENCH_RE = re.compile(
    r"^Benchmark(All|Protobuf)(Marshals|Unmarshals)_(\w+)/"
    r"(\w+)-\d+\s+"
    r"(\d+)\s+"
    r"([\d.]+)\s+ns/op"
    r"(?:\s+(\d+)\s+B/op)?"
    r"(?:\s+(\d+)\s+allocs/op)?",
)


def parse_bench_output(path):
    raw = defaultdict(list)
    with open(path) as f:
        for line in f:
            m = BENCH_RE.match(line.strip())
            if not m:
                continue
            op = m.group(2)       # Marshals / Unmarshals
            dtype = m.group(3)    # APIResponse, Player, ...
            codec = m.group(4)    # RAF, JSON, ...
            key = (op, dtype, codec)
            raw[key].append({
                "ns_op":  float(m.group(6)),
                "b_op":   int(m.group(7)) if m.group(6) else 0,
                "allocs": int(m.group(8)) if m.group(7) else 0,
            })

    results = []
    for (op, dtype, codec), runs in raw.items():
        n = len(runs)
        results.append({
            "op":     op.rstrip("s"),   # normalize to "Marshal" / "Unmarshal"
            "dtype":  dtype,
            "codec":  codec,
            "ns_op":  sum(r["ns_op"] for r in runs) / n,
            "b_op":   sum(r["b_op"] for r in runs) / n,
            "allocs": sum(r["allocs"] for r in runs) / n,
        })
    return results


CODEC_ORDER = ["RAF", "JSON", "MsgPack", "CBOR", "BSON"]


def _sorted_codecs(codecs):
    return sorted(codecs, key=lambda c: CODEC_ORDER.index(c) if c in CODEC_ORDER else 99)


def generate_summary(results, out_path):
    groups = defaultdict(list)
    for r in results:
        groups[(r["dtype"], r["op"])].append(r)

    lines = [
        "# RAF Benchmark Results\n",
        "",
    ]

    for (dtype, op), items in sorted(groups.items()):
        lines.append(f"### {dtype} - {op}\n")

        codecs = _sorted_codecs(list({r["codec"] for r in items}))
        lines.append("| Codec | ns/op | B/op | allocs/op |")
        lines.append("|-------|------:|-----:|----------:|")

        fastest_ns = min(r["ns_op"] for r in items)
        raf_ns = r["ns_op"] if (r := next((r for r in items if r["codec"] == "RAF"), None)) else None
        for codec in codecs:
            r = next((r for r in items if r["codec"] == codec), None)
            if not r:
                continue
            
            ratio = r["ns_op"] / raf_ns
            speedup = f" ({ratio:.1f}x)"

            codec_with_style = f"**{codec}**" if r["ns_op"] == fastest_ns else codec

            lines.append(
                f"| {codec_with_style} | {r['ns_op']:,.0f}{speedup} | {int(r['b_op']):,} | {int(r['allocs']):,} |"
            )
        lines.append("")

    with open(out_path, "w") as f:
        f.write("\n".join(lines))


def main():
    bench_file = "bench_raw.txt"

    results = parse_bench_output(bench_file)
    if not results:
        print(f"No benchmark results found in {bench_file}", file=sys.stderr)
        sys.exit(1)

    print(f"Parsed {len(results)} benchmark results")

    summary_path = "README.md"
    generate_summary(results, summary_path)
    print(f"Summary written to {summary_path}")


if __name__ == "__main__":
    main()
