workspace(name = "martian")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

RULES_GO_VERSION = "v0.21.0"

_RULES_GO_ARCHIVE = "github.com/bazelbuild/rules_go/releases/download/{}/rules_go-{}.tar.gz".format(
    RULES_GO_VERSION,
    RULES_GO_VERSION,
)

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "b27e55d2dcc9e6020e17614ae6e0374818a3e3ce6f2024036e688ada24110444",
    urls = [
        "https://storage.googleapis.com/bazel-mirror/" + _RULES_GO_ARCHIVE,
        "https://" + _RULES_GO_ARCHIVE,
    ],
)

GAZELLE_VERSION = "v0.19.1"

_GAZELLE_ARCHIVE = "github.com/bazelbuild/bazel-gazelle/releases/download/{}/bazel-gazelle-{}.tar.gz".format(
    GAZELLE_VERSION,
    GAZELLE_VERSION,
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "86c6d481b3f7aedc1d60c1c211c6f76da282ae197c3b3160f54bd3a8f847896f",
    urls = [
        "https://storage.googleapis.com/bazel-mirror/" + _GAZELLE_ARCHIVE,
        "https://" + _GAZELLE_ARCHIVE,
    ],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains()

load("//:deps.bzl", "martian_dependencies")

martian_dependencies()

load("@build_bazel_rules_nodejs//:index.bzl", "node_repositories")

node_repositories(package_json = ["//web/martian:package.json"])

load("//:npm.bzl", "martian_npm_repo")

martian_npm_repo()
