load("@bazel_gazelle//:deps.bzl", "go_repository")

def go_deps():
    go_repository(
        name = "com_github_osohq_go_oso",
        importpath = "github.com/osohq/go-oso",
        sum = "h1:QSgzYosP/yiaq6/YZu8DsA5Lzslo00bWcdOfQ3P4rn8=",
        version = "v0.26.1",
    )
