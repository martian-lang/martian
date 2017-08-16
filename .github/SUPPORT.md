# Support policy

Open source Martian is not officially supported by 10X Genomics.  If you
encounter problems running a 10X pipeline with an open source version of
Martian, please try with a Martian version which was validated with that
pipeline release before contacting
[10X genomics support](support@10xgenomics.com).

This is primarily a Linux product.  On OSX, we expect it to build, and we
expect the simpler tools like `mfc` and `mrc` to be functional.  We hope
for `mrp` to be able to run simple pipelines, but do not intend to put effort
into supporting them.  We'd like to get Windows support for `mrc`, `mrf`, and
possibly `mrp` in inspect-only mode working, but that is currently far from
ready.

That said, the code is the same as what ships (or will eventually ship) with
validated software releases, so we're happy to hear from users who want to
live on the bleeding edge about any bugs which are discovered through the
GitHub issue tracker.  Different tools have different reliability expectations:

* `mrp` is a core component and is expected to be stable in local mode, and
in SGE cluster mode.  Bugs in those modes are considered release blockers,
mostly, although the diversity of configuration options for SGE clusters means
in practice there are some configurations we can't properly support.
* `mrf`, `mrc`, `mrs` and so on should generally work, but only the most severe
bugs in them are considered to be release blockers.
* Issues with other cluster types, such as Slurm or LSF, are things we'd like
to improve on, but are not officially supported.  We welcome actionable
feedback on other cluster types.
