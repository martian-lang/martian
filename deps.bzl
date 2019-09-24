load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@bazel_gazelle//:deps.bzl", "go_repository")

def martian_dependencies():
    _maybe(
        go_repository,
        name = "com_github_cloudfoundry_gosigar",
        importpath = "github.com/cloudfoundry/gosigar",
        tag = "v1.1.0",
    )

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
        go_repository,
        name = "com_github_satori_go_uuid",
        commit = "0aa62d5ddceb",
        importpath = "github.com/satori/go.uuid",
    )

    _maybe(
        go_repository,
        name = "org_golang_x_sys",
        commit = "d0b11bdaac8adb652bff00e49bcacf992835621a",
        importpath = "golang.org/x/sys",
        shallow_since = "1547471016 +0000",
    )

    _maybe(
        # This actually already brought in by rules_go, and
        # is included here mostly for clarity.
        go_repository,
        name = "org_golang_x_tools",
        commit = "c8855242db9c1762032abe33c2dff50de3ec9d05",
        importpath = "golang.org/x/tools",
    )

    _maybe(
        http_archive,
        name = "build_bazel_rules_nodejs",
        sha256 = "88e5e579fb9edfbd19791b8a3c6bfbe16ae3444dba4b428e5efd36856db7cf16",
        urls = ["https://github.com/bazelbuild/rules_nodejs/releases/download/0.27.8/rules_nodejs-0.27.8.tar.gz"],
    )

def _maybe(repo_rule, name, **kwargs):
    if name not in native.existing_rules():
        repo_rule(name = name, **kwargs)
