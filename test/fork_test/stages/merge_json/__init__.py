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
    with open(args.json1, "r") as inp1:
        s1 = json.load(inp1)
    with open(args.json2, "r") as inp2:
        s2 = json.load(inp2)
    s1.update(s2)
    with open(outs.result, "w") as outf:
        json.dump(s1, outf, indent=2, sort_keys=True)
