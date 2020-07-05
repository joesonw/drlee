load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["plugins.go"],
    importpath = "github.com/joesonw/drlee/pkg/plugins",
    visibility = ["//visibility:public"],
    deps = [
        "//pkg/core:go_default_library",
        "@com_github_yuin_gopher_lua//:go_default_library",
    ],
)
