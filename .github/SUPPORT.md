# Support Policy

Open source Martian is a community project and is not officially supported
by 10x Genomics.  If you are running a 10x Genomics pipeline, please only
use the released, validated version of Martian that is bundled with that
pipeline.

Martian should be possible to build on both macOS and Windows.
MRO writing tools such as `mro` should work.
We would also like `mrp` to be able to run pipelines on those other operating
systems or architectures, but due to various differences between Linux and
these other operating systems, this has become increasingly challenging.
We would welcome contributions from the community to enable support for running
pipelines on Windows or macOS.

That said, the code is the same as what ships with validated software releases,
so we are happy to hear from users of bleeding edge open source builds about
any bugs discovered. Please file them using the GitHub issue tracker.  
Different tools in Martian have different release qualification expectations:

* `mrp` is a core component and is expected to be stable in local mode, and
in SGE cluster mode.  Bugs in those modes are considered release blockers,
although the diversity of configuration options for SGE clusters means
in practice there are some configurations that cannot be properly supported.
* Issues with other cluster types, such as Slurm or LSF, are things we would
like to improve on, but are not officially supported.  We welcome actionable
feedback on these and other cluster types.
* `mro` are also expected to be stable.
