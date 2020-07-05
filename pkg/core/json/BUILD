load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = ["json.go"],
    importpath = "github.com/joesonw/drlee/pkg/core/json",
    visibility = ["//visibility:public"],
    deps = [
        "//pkg/utils:go_default_library",
        "@com_github_yuin_gopher_lua//:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = ["json_test.go"],
    embed = [":go_default_library"],
    deps = [
        "//pkg/core/test:go_default_library",
        "@com_github_onsi_ginkgo//:go_default_library",
        "@com_github_onsi_gomega//:go_default_library",
        "@com_github_yuin_gopher_lua//:go_default_library",
    ],
)
