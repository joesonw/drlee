load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "client.go",
        "conn.go",
        "network.go",
        "server.go",
    ],
    importpath = "github.com/joesonw/drlee/pkg/core/network",
    visibility = ["//visibility:public"],
    deps = [
        "//pkg/core:go_default_library",
        "//pkg/core/helpers/params:go_default_library",
        "//pkg/core/object:go_default_library",
        "//pkg/core/stream:go_default_library",
        "//pkg/utils:go_default_library",
        "@com_github_yuin_gopher_lua//:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = [
        "client_test.go",
        "network_test.go",
        "server_test.go",
    ],
    embed = [":go_default_library"],
    deps = [
        "//pkg/core:go_default_library",
        "//pkg/core/test:go_default_library",
        "@com_github_onsi_ginkgo//:go_default_library",
        "@com_github_onsi_gomega//:go_default_library",
        "@com_github_yuin_gopher_lua//:go_default_library",
    ],
)
