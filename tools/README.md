# Martian tools

This directory contains developer tools for working with martian.

## Editor syntax highlighting

The [syntax](syntax) directory contains syntax highlighting rules for various
editors.  The level of support varies, as does the method of installation.

The syntax highlighting [grammar definition][] shared between `atom`, `sublime`,
and Visual Studio Code (or any other editor which can consume TextMate grammars)
is probably the most up to date and complete, followed closely by the
[one for vim](syntax/vim/martian.vim).  There is bare-bones support for
[pycharm](syntax/pycharm/martian.xml) as well.  Contributions adding or
improving support are of course welcome other editors.

[grammar definition]: syntax/sublimetext/Martianlang.YAML-tmLanguage

Setting up your editor to use the syntax highlighting files of course varies
between editors.  For Visual Studio Code, build the extension with
`make vscode` in the root, and put a symlink from
`~/.vscode/extensions/martian-<version>`
(or `~/.vscode-server/extensions/martian-<version>` for remote mode) to the
`tools/syntax/vscode` of the repository, or use the `vsce` tool to package it.
For vim, add `syntax/vim/martian.vim` to `~/.vim/syntax` and `~/.vim/ftdetect`.

## Bazel rules

When building projects with [bazel](https://bazel.build), the rules found in
[`mro_rules.bzl`](mro_rules.bzl) will be useful.  Documentation for the primary
rules can be found in [`docs/mro_rules.md`](docs/mro_rules.md).
