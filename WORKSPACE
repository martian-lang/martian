workspace(name = "martian")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

RULES_GO_VERSION = "v0.48.1"

_RULES_GO_ARCHIVE = "github.com/bazelbuild/rules_go/releases/download/{}/rules_go-{}.zip".format(
    RULES_GO_VERSION,
    RULES_GO_VERSION,
)

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "b2038e2de2cace18f032249cb4bb0048abf583a36369fa98f687af1b3f880b26",
    urls = [
        "https://mirror.bazel.build/" + _RULES_GO_ARCHIVE,
        "https://" + _RULES_GO_ARCHIVE,
    ],
)

GAZELLE_VERSION = "v0.37.0"

_GAZELLE_ARCHIVE = "github.com/bazelbuild/bazel-gazelle/releases/download/{}/bazel-gazelle-{}.tar.gz".format(
    GAZELLE_VERSION,
    GAZELLE_VERSION,
)

http_archive(
    name = "bazel_gazelle",
    integrity = "sha256-12v3pg/YsFBEQJDfooN6Tq+YKeEWVhjuNdzspcvfWNU=",
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

# Required for bazel 8.  Assume dependent projects are bringing their own.
http_archive(
    name = "rules_cc",
    sha256 = "abc605dd850f813bb37004b77db20106a19311a96b2da1c92b789da529d28fe1",
    strip_prefix = "rules_cc-0.0.17",
    urls = ["https://github.com/bazelbuild/rules_cc/releases/download/0.0.17/rules_cc-0.0.17.tar.gz"],
)

load("@bazel_skylib//:workspace.bzl", "bazel_skylib_workspace")

bazel_skylib_workspace()

# Development only, not required by dependent projects:

http_archive(
    name = "io_bazel_stardoc",
    sha256 = "fabb280f6c92a3b55eed89a918ca91e39fb733373c81e87a18ae9e33e75023ec",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/stardoc/releases/download/0.7.1/stardoc-0.7.1.tar.gz",
        "https://github.com/bazelbuild/stardoc/releases/download/0.7.1/stardoc-0.7.1.tar.gz",
    ],
)

protobuf_tag = "27.5"

http_archive(
    name = "com_google_protobuf",
    sha256 = "79cc6d09d02706c5a73e900ea842b5b3dae160f371b6654774947fe781851423",
    strip_prefix = "protobuf-" + protobuf_tag,
    urls = [
        "https://github.com/protocolbuffers/protobuf/releases/download/v{}/protobuf-{}.tar.gz".format(
            protobuf_tag,
            protobuf_tag,
        ),
    ],
)

load("@io_bazel_stardoc//:setup.bzl", "stardoc_repositories")

stardoc_repositories()

load("@bazel_features//:deps.bzl", "bazel_features_deps")

bazel_features_deps()

load("@rules_python//python:repositories.bzl", "py_repositories")

py_repositories()

load("@rules_jvm_external//:repositories.bzl", "rules_jvm_external_deps")

rules_jvm_external_deps()

load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")

protobuf_deps()

load("@rules_jvm_external//:setup.bzl", "rules_jvm_external_setup")

rules_jvm_external_setup()

load("@io_bazel_stardoc//:deps.bzl", "stardoc_external_deps")

stardoc_external_deps()

load("@stardoc_maven//:defs.bzl", stardoc_pinned_maven_install = "pinned_maven_install")

stardoc_pinned_maven_install()
