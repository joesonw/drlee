load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "async.go",
        "lib.go",
    ],
    importpath = "github.com/joesonw/drlee/pkg/lib",
    visibility = ["//visibility:public"],
    deps = ["@com_github_yuin_gopher_lua//:go_default_library"],
)

go_test(
    name = "go_default_test",
    srcs = [
        "async_test.go",
        "lib_test.go",
    ],
    embed = [":go_default_library"],
    deps = [
        "//pkg/builtin:go_default_library",
        "@com_github_onsi_ginkgo//:go_default_library",
        "@com_github_onsi_gomega//:go_default_library",
        "@com_github_yuin_gopher_lua//:go_default_library",
    ],
)
