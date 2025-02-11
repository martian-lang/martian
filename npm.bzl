""" Workspace macro to load martian's npm repositories. """

load("@build_bazel_rules_nodejs//:index.bzl", "yarn_install")

def martian_npm_repo():
    """ Workspace macro to load martian's npm repositories. """
    yarn_install(
        name = "martian_npm",
        package_json = "@martian//web/martian:package.json",
        yarn_lock = "@martian//web/martian:yarn.lock",
        symlink_node_modules = False,
        args = [
            "--frozen-lockfile",
            "--ignore-optional",
            "--mutex",
            "network",
        ],
    )
