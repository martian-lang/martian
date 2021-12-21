"""Repository macro to load remote dependencies."""

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@bazel_tools//tools/build_defs/repo:utils.bzl", "maybe")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

def martian_dependencies(
        rules_nodejs_version = "4.5.1",
        rules_nodejs_sha = "f0f76a06fd6c10e8fb9a6cb9389fa7d5816dbecd9b1685063f89fb20dc6822f3"):
    """Loads remote repositories required to build martian.

    Args:
        rules_nodejs_version: Override the default version of rules_nodejs.
        rules_nodejs_sha: Override the expected checksum for rules_nodejs.
    """

    # Do this before gazelle_dependencies because gazelle wants
    # an older version.
    # This should actually already have been brought in by rules_go, but is
    # added here for clarity.
    maybe(
        http_archive,
        name = "bazel_skylib",
        urls = [
            "https://mirror.bazel.build/github.com/bazelbuild/bazel-skylib/releases/download/1.1.1/bazel-skylib-1.1.1.tar.gz",
            "https://github.com/bazelbuild/bazel-skylib/releases/download/1.1.1/bazel-skylib-1.1.1.tar.gz",
        ],
        sha256 = "c6966ec828da198c5d9adbaa94c05e3a1c7f21bd012a0b29ba8ddbccb2c93b0d",
    )

    # Also do this before gazelle_dependencies.
    maybe(
        go_repository,
        name = "org_golang_x_sys",
        commit = "9d821ace865463a58a9957ad908018bd0d005f48",
        importpath = "golang.org/x/sys",
    )

    gazelle_dependencies()

    maybe(
        go_repository,
        name = "com_github_dustin_go_humanize",
        commit = "9f541cc9db5d55bce703bd99987c9d5cb8eea45e",
        importpath = "github.com/dustin/go-humanize",
    )

    maybe(
        go_repository,
        name = "com_github_google_shlex",
        commit = "e7afc7fbc51079733e9468cdfd1efcd7d196cd1d",
        importpath = "github.com/google/shlex",
    )

    maybe(
        go_repository,
        name = "com_github_martian_lang_docopt_go",
        commit = "57cc8f5f669dae55ae1beb7a6310ea2f58ea61d5",
        importpath = "github.com/martian-lang/docopt.go",
    )

    maybe(
        # This actually already brought in by rules_go, and
        # is included here mostly for clarity.
        go_repository,
        name = "org_golang_x_tools",
        commit = "e212aff8fd146c44ddb0167c1dfbd5531d6c9213",
        importpath = "golang.org/x/tools",
    )

    maybe(
        http_archive,
        name = "build_bazel_rules_nodejs",
        sha256 = rules_nodejs_sha,
        urls = [
            "https://github.com/bazelbuild/rules_nodejs/releases/download/" +
            "{}/rules_nodejs-{}.tar.gz".format(
                rules_nodejs_version,
                rules_nodejs_version,
            ),
        ],
    )

    python_rules_commit = "c8c79aae9aa1b61d199ad03d5fe06338febd0774"
    maybe(
        http_archive,
        name = "rules_python",
        sha256 = "95ee649313caeb410b438b230f632222fb5d2053e801fe4ae0572eb1d71e95b8",
        strip_prefix = "rules_python-" + python_rules_commit,
        urls = [
            "https://github.com/bazelbuild/rules_python/archive/{}.tar.gz".format(
                python_rules_commit,
            ),
        ],
    )
