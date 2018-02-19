This directory contains only symlinks required for the Makefile.  The preferred
way to check out the Martian repo is into the
$GOPATH/github.com/martian-lang/martian
directory.  However, this directory and the symlinks under it allow the
Makefile to set $GOPATH to the repository root in case it was checked out
somewhere else.  Do not put source code here, as it is unlikely to be found
correctly in cases where the repository is cloned into GOPATH as a library.
