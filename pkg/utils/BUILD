load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "gob.go",
        "lua.go",
        "pprof.go",
    ],
    importpath = "github.com/joesonw/drlee/pkg/utils",
    visibility = ["//visibility:public"],
    deps = [
        "@com_github_yuin_gopher_lua//:go_default_library",
        "@org_uber_go_zap//:go_default_library",
    ],
)
