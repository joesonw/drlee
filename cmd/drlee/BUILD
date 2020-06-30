load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    importpath = "github.com/joesonw/drlee/cmd/drlee",
    visibility = ["//visibility:private"],
    deps = [
        "//cmd/drlee/commands:go_default_library",
        "//pkg/utils:go_default_library",
        "@com_github_spf13_cobra//:go_default_library",
        "@org_uber_go_zap//:go_default_library",
    ],
)

go_binary(
    name = "drlee",
    embed = [":go_default_library"],
    visibility = ["//visibility:public"],
)
