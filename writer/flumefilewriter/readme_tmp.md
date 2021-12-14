# flumefilewriter

## 使用该writer需要的目录结构

```

/data/
├── flume_test
│   ├── 1
│   │   ├── multiplexing
│   │   └── replicating
│   └── 2
│       ├── multiplexing
│       └── replicating
└── temp
    ├── multiplexing
    └── replicating

```
`/data/flume_test` 为日志文件记录根地址`RootPath`配置，
内部需要有`1`、`2`等分片目录，分片目录中的`replicating`和`multiplexing`对应writer的`selectorType`配置。

`/data/temp` 为日志文件记录临时地址`tempFilePath`配置；
分片目录中的`replicating`和`multiplexing`对应writer的`selectorType`配置。

在正式环境中，分片目录，`replicating`目录会由`flume`日志维护人员提前创建好。


