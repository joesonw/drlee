load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "a_runtime-packr.go",
        "runtime.go",
    ],
    importpath = "github.com/joesonw/drlee/pkg/runtime",
    visibility = ["//visibility:public"],
    deps = ["@com_github_gobuffalo_packr//:go_default_library"],
)
