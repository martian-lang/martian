import martian
import json

__MRO__ = """
stage MERGE_JSON(
    in json json1,
    in json json2,
    out json result,
    src py "stages/merge_json",
)
"""


def main(args, outs):
    result = {}
    for fname in args.inputs:
        with open(fname, 'r') as inp:
            result.update(json.load(inp))
    with open(outs.result, 'w') as outf:
        json.dump(result, outf, indent=2, sort_keys=True)
