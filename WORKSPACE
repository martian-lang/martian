workspace(name = "martian")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

RULES_GO_VERSION = "v0.42.0"

_RULES_GO_ARCHIVE = "github.com/bazelbuild/rules_go/releases/download/{}/rules_go-{}.zip".format(
    RULES_GO_VERSION,
    RULES_GO_VERSION,
)

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "91585017debb61982f7054c9688857a2ad1fd823fc3f9cb05048b0025c47d023",
    urls = [
        "https://mirror.bazel.build/" + _RULES_GO_ARCHIVE,
        "https://" + _RULES_GO_ARCHIVE,
    ],
)

GAZELLE_VERSION = "v0.32.0"

_GAZELLE_ARCHIVE = "github.com/bazelbuild/bazel-gazelle/releases/download/{}/bazel-gazelle-{}.tar.gz".format(
    GAZELLE_VERSION,
    GAZELLE_VERSION,
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "29218f8e0cebe583643cbf93cae6f971be8a2484cdcfa1e45057658df8d54002",
    urls = [
        "https://storage.googleapis.com/bazel-mirror/" + _GAZELLE_ARCHIVE,
        "https://" + _GAZELLE_ARCHIVE,
    ],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains(version = "host")

load("//:deps.bzl", "martian_dependencies")

# gazelle:repository_macro deps.bzl%martian_dependencies
martian_dependencies()

load("@build_bazel_rules_nodejs//:index.bzl", "node_repositories")

node_repositories(package_json = ["//web/martian:package.json"])

load("//:npm.bzl", "martian_npm_repo")

martian_npm_repo()

# Development only, not required by dependent projects:

http_archive(
    name = "io_bazel_stardoc",
    sha256 = "05fb57bb4ad68a360470420a3b6f5317e4f722839abc5b17ec4ef8ed465aaa47",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/stardoc/releases/download/0.5.2/stardoc-0.5.2.tar.gz",
        "https://github.com/bazelbuild/stardoc/releases/download/0.5.2/stardoc-0.5.2.tar.gz",
    ],
)
