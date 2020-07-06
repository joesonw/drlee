load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "conn.go",
        "server.go",
        "websocket.go",
    ],
    importpath = "github.com/joesonw/drlee/pkg/core/websocket",
    visibility = ["//visibility:public"],
    deps = [
        "//pkg/core:go_default_library",
        "//pkg/core/object:go_default_library",
        "//pkg/core/stream:go_default_library",
        "//pkg/utils:go_default_library",
        "@com_github_gobwas_ws//:go_default_library",
        "@com_github_gobwas_ws//wsutil:go_default_library",
        "@com_github_yuin_gopher_lua//:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = [
        "server_test.go",
        "websocket_test.go",
    ],
    embed = [":go_default_library"],
    deps = [
        "//pkg/core:go_default_library",
        "//pkg/core/test:go_default_library",
        "@com_github_gobwas_ws//:go_default_library",
        "@com_github_gobwas_ws//wsutil:go_default_library",
        "@com_github_onsi_ginkgo//:go_default_library",
        "@com_github_onsi_gomega//:go_default_library",
        "@com_github_yuin_gopher_lua//:go_default_library",
    ],
)
