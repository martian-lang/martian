# Support Policy

Open source Martian is a community project and is not officially supported
by 10x Genomics.  If you are running a 10x Genomics pipeline, please only
use the released, validated version of Martian that is bundled with that
pipeline.

Martian is primarily an `x86_64` Linux product. However, it was originally
developed on 64-bit macOS, and we intend for Martian to build on macOS, and
for MRO writing tools such as `mrc` and `mrf` to work. We would also like
`mrp` to be able to run pipelines, but due to growing differences between
macOS and Linux, this could become increasingly challenging. We would also
like to see contributions from the community to enable support on Windows.

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
* `mrc`, `mrf`, and `mrs` are also expected to be stable.
