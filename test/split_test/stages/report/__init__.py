# encoding: utf-8

import martian

__MRO__ = """
stage REPORT(
    in  float[] values,
    in  float   sum,
)
"""

def main(args, outs):
    martian.log_info(u'Logging à non-ascii character.')
    martian.update_progress(u'%s = %f' % (
        u'+'.join([u'%f\u00b2' % v for v in args.values]),
        args.sum))
