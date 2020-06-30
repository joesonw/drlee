package server

import "time"

type Config struct {
	NodeName    string       `yaml:"node-name"`
	Gossip      GossipConfig `yaml:"gossip"`
	RPC         RPCConfig    `yaml:"rpc"`
	Concurrency int          `yaml:"concurrency"`
	Queue       QueueConfig  `yaml:"queue"`
}

type QueueConfig struct {
	Dir             string        `yaml:"dir"`
	MaxBytesPerFile int64         `yaml:"max-bytes-per-file"`
	MaxMsgSize      int32         `yaml:"max-msg-size"`
	SyncEvery       int64         `yaml:"sync-every"`
	SyncTimeout     time.Duration `yaml:"sync-timeout"`
}

type GossipConfig struct {
	Addr      string `yaml:"addr"`
	Port      int    `yaml:"port"`
	SecretKey string `yaml:"secret-key"`
}

type RPCConfig struct {
	Addr             string `yaml:"addr"`
	Port             int32  `yaml:"port"`
	ReplyConcurrency int    `yaml:"reply-concurrency"`
}

type ScriptConfig struct {
	File        string `yaml:"file"`
	Concurrency int    `yaml:"concurrency"`
}
