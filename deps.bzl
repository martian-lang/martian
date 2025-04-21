"""Repository macro to load remote dependencies."""

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@bazel_tools//tools/build_defs/repo:utils.bzl", "maybe")

def martian_dependencies(
        rules_nodejs_version = "4.7.0",
        rules_nodejs_sha = "6f15d75f9e99c19d9291ff8e64e4eb594a6b7d25517760a75ad3621a7a48c2df"):
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
            "https://mirror.bazel.build/github.com/bazelbuild/bazel-skylib/releases/download/1.7.1/bazel-skylib-1.7.1.tar.gz",
            "https://github.com/bazelbuild/bazel-skylib/releases/download/1.7.1/bazel-skylib-1.7.1.tar.gz",
        ],
        sha256 = "bc283cdfcd526a52c3201279cda4bc298652efa898b10b4db0837dc51652756f",
    )

    # Also do this before gazelle_dependencies.
    maybe(
        go_repository,
        name = "org_golang_x_sys",
        # v0.20.0, latest as of 2024-05-20
        version = "v0.30.0",
        sum = "h1:QjkSwP/36a20jFYWkSue1YwXzLmsV5Gfq7Eiy72C1uc=",
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
        # v0.21.0, latest as of 2024-05-20
        version = "v0.30.0",
        importpath = "golang.org/x/tools",
        sum = "h1:BgcpHewrV5AUp2G9MebG4XPFI1E2W41zU1SaqVA9vJY=",
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

    python_rules_tag = "0.30.0"
    python_rules_sha = "3b8b4cdc991bc9def8833d118e4c850f1b7498b3d65d5698eea92c3528b8cf2c"
    maybe(
        http_archive,
        name = "rules_python",
        sha256 = python_rules_sha,
        strip_prefix = "rules_python-" + python_rules_tag,
        urls = [
            "https://github.com/bazelbuild/rules_python/archive/refs/tags/{}.tar.gz".format(
                python_rules_tag,
            ),
        ],
    )
