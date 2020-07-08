load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["rpc.pb.go"],
    importpath = "github.com/joesonw/drlee/dist/bin/_proto/proto_go_proto_/github.com/joesonw/drlee/proto",
    visibility = ["//visibility:public"],
    deps = [
        "@com_github_golang_protobuf//proto:go_default_library",
        "@org_golang_google_grpc//:go_default_library",
        "@org_golang_google_grpc//codes:go_default_library",
        "@org_golang_google_grpc//status:go_default_library",
        "@org_golang_google_protobuf//reflect/protoreflect:go_default_library",
        "@org_golang_google_protobuf//runtime/protoimpl:go_default_library",
    ],
)
