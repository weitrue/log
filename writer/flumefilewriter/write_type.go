package flumefilewriter

// 指包含分片目录的根目录
// const RootPath = "/data/flume_test"

// 如果分片目录大于1000,log直接存入该文件中
// const MuchInforFile = "C:/Users/Administrator/Desktop/temp/123"

// SendMode 日记发送方式：multiplexing or replicating
type SendMode int

const (
	Multiplexing SendMode = iota
	Replicating
)

func (s SendMode) toString() string {
	switch s {
	// case Multiplexing:
	//	return "multiplexing"
	case Replicating:
		return "replicating"
	default:
		return "multiplexing"
	}
}

// SelectorType 文件将要发送的集群或分区
type SelectorType int

const (
	ES SelectorType = iota
	HDFS1
	HDFS2
)
