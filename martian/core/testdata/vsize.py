"""This little script is intended for testing.  It allocates
1GB of anonymous virtual address space, then closes its standard
output pipe and waits for its standard input pipe to close."""

import mmap
import os
import sys


def main(argv):
    """Reserve argv[1] of virtual address space, then consume argv[2] of rss,
    then close standard output and wait until standard input is closed."""
    vmem_kb = int(argv[1])
    rss_kb = int(argv[2])

    vmem = mmap.mmap(-1, vmem_kb*1024, access=mmap.ACCESS_WRITE)
    kb_string = b'01234567'*128
    for _ in range(rss_kb):
        vmem.write(kb_string)

    # Close stdout to signal readiness to the parent process.
    os.close(1)

    for _ in sys.stdin:
        pass


if __name__ == '__main__':
    main(sys.argv)
