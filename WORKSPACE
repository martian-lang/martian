workspace(name = "martian")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

RULES_GO_VERSION = "v0.22.4"

_RULES_GO_ARCHIVE = "github.com/bazelbuild/rules_go/releases/download/{}/rules_go-{}.tar.gz".format(
    RULES_GO_VERSION,
    RULES_GO_VERSION,
)

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "7b9bbe3ea1fccb46dcfa6c3f3e29ba7ec740d8733370e21cdc8937467b4a4349",
    urls = [
        "https://storage.googleapis.com/bazel-mirror/" + _RULES_GO_ARCHIVE,
        "https://" + _RULES_GO_ARCHIVE,
    ],
)

GAZELLE_VERSION = "v0.20.0"

_GAZELLE_ARCHIVE = "github.com/bazelbuild/bazel-gazelle/releases/download/{}/bazel-gazelle-{}.tar.gz".format(
    GAZELLE_VERSION,
    GAZELLE_VERSION,
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "d8c45ee70ec39a57e7a05e5027c32b1576cc7f16d9dd37135b0eddde45cf1b10",
    urls = [
        "https://storage.googleapis.com/bazel-mirror/" + _GAZELLE_ARCHIVE,
        "https://" + _GAZELLE_ARCHIVE,
    ],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains()

load("//:deps.bzl", "martian_dependencies")

# gazelle:repository_macro deps.bzl%martian_dependencies
martian_dependencies()

load("@build_bazel_rules_nodejs//:index.bzl", "node_repositories")

node_repositories(package_json = ["//web/martian:package.json"])

load("//:npm.bzl", "martian_npm_repo")

martian_npm_repo()

# Development only, not required by dependent projects:

_STARDOC_COMMIT = "8d076b25d4f66a7a37125f62dab6357e4d46603e"

http_archive(
    name = "io_bazel_stardoc",
    sha256 = "d3b602c511a1a918953c0458cf93ee22a6467ed51aa4c24cc13cd6fd48b27dac",
    strip_prefix = "stardoc-" + _STARDOC_COMMIT,
    urls = [
        "https://github.com/bazelbuild/stardoc/archive/{}.tar.gz".format(_STARDOC_COMMIT),
    ],
)
