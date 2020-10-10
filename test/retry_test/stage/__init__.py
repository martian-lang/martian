""" This is stage code for the autoretry test.

The code is intended to fail on the first try and succeed
on later retries.  Because of this, it does things that
stage code generally should not be doing, so do not take
anything in here to be good practice for normal stage code.

This stage can be run either as chunks or as split/chunk/join.
"""

import os
import martian

__MRO__ = """
stage BEGIN(
    in  int    count,
    out file[] sentinels,
    out bool[] should_fail_next,
    src py     "stage",
) split (
    in  file   sentinel,
    in  bool   should_fail,
    out file   sentinel,
    out bool   should_fail,
) using (
    volatile = strict,
)

stage MAYBE_FAIL(
    in  file sentinel,
    in  bool should_fail,
    out file sentinel,
    out bool should_fail,
    src py   "stage",
) using (
    volatile = strict,
)
"""


def split(args):
    """ Creates args.count chunks, and sets up the first one fail. """
    sentinel = martian.make_path("sentinel")
    with open(sentinel, "w") as sentinel_file:
        sentinel_file.write("fail this attempt")
    return {
        "chunks": [
            {"sentinel": sentinel if not i else "", "should_fail": not i,}
            for i in range(args.count)
        ]
    }


def main(args, outs):
    """ Fails if args.sentinel is non-empty and refers to a file that exists. """
    sentinel = ""
    if args.should_fail:
        sentinel = martian.make_path("sentinel")
        with open(sentinel, "w") as sentinel_file:
            sentinel_file.write("fail this attempt")
    outs.should_fail = args.should_fail
    outs.sentinel = sentinel
    if args.sentinel:
        # Do not do this in normal stage code.  Stage code should not modify its
        # inputs.  The reason for it working this way here is to have the stage fail
        # on the first try but not the second.
        try:
            os.unlink(args.sentinel)
        except OSError:
            pass
        else:
            # Use an error message which will trigger auto-retry.
            martian.throw("resource temporarily unavailable")


def join(args, outs, chunk_defs, chunk_outs):
    """ Collects the chunk outputs. """
    # pylint: disable=unused-argument
    outs.should_fail_next = [chunk.should_fail for chunk in chunk_outs]
    outs.sentinels = [chunk.sentinel for chunk in chunk_outs]
