load("@build_bazel_rules_nodejs//:index.bzl", "npm_install")

def martian_npm_repo():
    npm_install(
        name = "martian_npm",
        package_json = "@martian//web/martian:package.json",
        package_lock_json = "@martian//web/martian:package-lock.json",
        symlink_node_modules = False,
        args = ["--frozen-lockfile"],
    )
