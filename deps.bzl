"""Repository macro to load remote dependencies."""

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@bazel_tools//tools/build_defs/repo:utils.bzl", "maybe")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

def martian_dependencies(
        rules_nodejs_version = "1.6.0",
        rules_nodejs_sha = "f9e7b9f42ae202cc2d2ce6d698ccb49a9f7f7ea572a78fd451696d03ef2ee116"):
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
        commit = "fde4db37ae7ad8191b03d30d27f258b5291ae4e3",
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
        commit = "65e3620a7ae7ac25e8494a60f0e5ef4e4fba03b3",
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
