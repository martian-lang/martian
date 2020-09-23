"""Repository macro to load remote dependencies."""

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@bazel_tools//tools/build_defs/repo:utils.bzl", "maybe")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

def martian_dependencies(
        rules_nodejs_version = "1.6.1",
        rules_nodejs_sha = "d14076339deb08e5460c221fae5c5e9605d2ef4848eee1f0c81c9ffdc1ab31c1"):
    """Loads remote repositories required to build martian.

    Args:
        rules_nodejs_version: Override the default version of rules_nodejs.
        rules_nodejs_sha: Override the expected checksum for rules_nodejs.
    """

    # Do this before gazelle_dependencies because gazelle wants
    # an older version.
    maybe(
        go_repository,
        name = "org_golang_x_sys",
        commit = "bc7a7d42d5c30f4d0fe808715c002826ce2c624e",
        importpath = "golang.org/x/sys",
    )

    # Also do this before gazelle_dependencies, which pulls version 0.5.0.
    maybe(
        http_archive,
        name = "bazel_skylib",
        urls = [
            "https://mirror.bazel.build/github.com/bazelbuild/bazel-skylib/releases/download/1.0.2/bazel-skylib-1.0.2.tar.gz",
            "https://github.com/bazelbuild/bazel-skylib/releases/download/1.0.2/bazel-skylib-1.0.2.tar.gz",
        ],
        sha256 = "97e70364e9249702246c0e9444bccdc4b847bed1eb03c5a3ece4f83dfe6abc44",
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
        commit = "c024452afbcdebb4a0fbe1bb0eaea0d2dbff835b",
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

    python_rules_commit = "a0fbf98d4e3a232144df4d0d80b577c7a693b570"
    maybe(
        http_archive,
        name = "rules_python",
        sha256 = "76a8fd4e7eca2a3590f816958faa0d83c9b2ce9c32634c5c375bcccf161d3bb5",
        strip_prefix = "rules_python-" + python_rules_commit,
        urls = [
            "https://github.com/bazelbuild/rules_python/archive/{}.tar.gz".format(
                python_rules_commit,
            ),
        ],
    )
