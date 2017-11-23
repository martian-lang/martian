import martian

__MRO__ = """
stage REPORT(
    in  float[] values,
    in  float   sum,
)
"""

def main(args, outs):
    if not args.sum is None:
        martian.update_progress('%s = %f' % (
            '+'.join(['%f^2' % v for v in args.values]),
            args.sum))
