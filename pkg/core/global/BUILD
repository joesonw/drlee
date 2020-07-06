load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["global.go"],
    importpath = "github.com/joesonw/drlee/pkg/core/global",
    visibility = ["//visibility:public"],
    deps = [
        "//pkg/core:go_default_library",
        "//pkg/utils:go_default_library",
        "@com_github_satori_go_uuid//:go_default_library",
        "@com_github_yuin_gopher_lua//:go_default_library",
    ],
)
