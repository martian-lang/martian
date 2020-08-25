"""Stage to compute the sum of values in an array"""

__MRO__ = """
stage SUM(
    in  float[] x,
    out float   sum,
    src py      "stages/sum",
)
"""


def main(args, outs):
    """Computes the sum of values in an array"""
    outs.sum = sum(args.x)
