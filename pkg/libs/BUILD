load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "all.go",
        "call.go",
        "callback.go",
        "check.go",
        "context.go",
        "env.go",
        "error.go",
        "go_object.go",
        "http.go",
        "http_server.go",
        "json.go",
        "log.go",
        "readonly.go",
        "redis.go",
        "regiser_global.go",
        "rpc.go",
        "sql.go",
        "table.go",
        "time.go",
    ],
    importpath = "github.com/joesonw/drlee/pkg/libs",
    visibility = ["//visibility:public"],
    deps = [
        "@com_github_go_redis_redis_v8//:go_default_library",
        "@com_github_satori_go_uuid//:go_default_library",
        "@com_github_yuin_gopher_lua//:go_default_library",
        "@org_uber_go_zap//:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = [
        "go_object_test.go",
        "http_server_test.go",
        "http_test.go",
        "json_test.go",
        "libs_suite_test.go",
        "sql_test.go",
        "table_test.go",
        "time_test.go",
    ],
    embed = [":go_default_library"],
    deps = [
        "@com_github_data_dog_go_sqlmock//:go_default_library",
        "@com_github_onsi_ginkgo//:go_default_library",
        "@com_github_onsi_gomega//:go_default_library",
        "@com_github_stretchr_testify//assert:go_default_library",
        "@com_github_yuin_gopher_lua//:go_default_library",
        "@ke_bou_monkey//:go_default_library",
    ],
)
