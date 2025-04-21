""" This is stage code for the autoretry test (map call over maps variant).

The code is intended to fail on the first try and succeed
on later retries.  Because of this, it does things that
stage code generally should not be doing, so do not take
anything in here to be good practice for normal stage code.

This stage can be run either as chunks or as split/chunk/join.
"""

import os
import martian


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
    # We should never have any files in the outs; if there is one, it implies that
    # we failed to clean up correctly.
    assert outs.sentinel
    assert not os.path.exists(outs.sentinel)
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
    outs.should_fail_next = {f"test{i}": chunk.should_fail for (i, chunk) in enumerate(chunk_outs)}
    outs.sentinels = {f"test{i}": chunk.sentinel for (i, chunk) in enumerate(chunk_outs)}
