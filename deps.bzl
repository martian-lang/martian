"""Repository macro to load remote dependencies."""

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

def martian_dependencies(
        rules_nodejs_version = "0.42.1",
        rules_nodejs_sha = "c612d6b76eaa17540e8b8c806e02701ed38891460f9ba3303f4424615437887a"):
    """Loads remote repositories required to build martian.

    Args:
        rules_nodejs_version: Override the default version of rules_nodejs.
        rules_nodejs_sha: Override the expected checksum for rules_nodejs.
    """

    # Do this before gazelle_dependencies because gazelle wants
    # an older version.
    _maybe(
        go_repository,
        name = "org_golang_x_sys",
        commit = "fde4db37ae7ad8191b03d30d27f258b5291ae4e3",
        importpath = "golang.org/x/sys",
    )

    gazelle_dependencies()

    _maybe(
        go_repository,
        name = "com_github_dustin_go_humanize",
        commit = "9f541cc9db5d",
        importpath = "github.com/dustin/go-humanize",
    )

    _maybe(
        go_repository,
        name = "com_github_google_shlex",
        commit = "6f45313302b9",
        importpath = "github.com/google/shlex",
    )

    _maybe(
        go_repository,
        name = "com_github_martian_lang_docopt_go",
        commit = "57cc8f5f669d",
        importpath = "github.com/martian-lang/docopt.go",
    )

    _maybe(
        # This actually already brought in by rules_go, and
        # is included here mostly for clarity.
        go_repository,
        name = "org_golang_x_tools",
        commit = "65e3620a7ae7ac25e8494a60f0e5ef4e4fba03b3",
        importpath = "golang.org/x/tools",
    )

    _maybe(
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

def _maybe(repo_rule, name, **kwargs):
    if name not in native.existing_rules():
        repo_rule(name = name, **kwargs)
