import martian

__MRO__ = """
stage REPORT(
    in  float[] values,
    in  float   sum,
)
"""

def main(args, outs):
    martian.update_progress('%s = %f' % (
        '+'.join(['%f^2' % v for v in args.values]),
        args.sum))
