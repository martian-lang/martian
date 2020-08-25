"""Trivial stage to compute z = x*y"""

__MRO__ = """
stage MULTIPLY(
    in  float x,
    in  float y,
    out float z,
    src py    "stages/multiply",
)
"""


def main(args, outs):
    """Computes z = x*y"""
    outs.product = args.x * args.y
