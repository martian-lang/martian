"""Stage to generate compute values = [being,end)"""

__MRO__ = """
stage RANGE(
    in  float begin,
    in  float end,
    out int[] values,
    src py    "stages/range",
)
"""

def main(args, outs):
    """Creates an array of integers from begin to end."""
    outs.values = list(range(int(args.begin), int(args.end)))