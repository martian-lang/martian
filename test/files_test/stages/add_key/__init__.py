import martian
import json
import os
import random

__MRO__ = """
stage ADD_KEY(
    in string key,
    in string value,
    in json start,
    in string failfile,
    out json result,
    src py "stages/add_key",
)
"""


def main(args, outs):
    if args.start:
        with open(args.start, 'r') as inpf:
            s = json.load(inpf)
    else:
        s = {}
    s[args.key] = args.value
    with open(outs.result, 'w') as outf:
        json.dump(s, outf, indent=2)
