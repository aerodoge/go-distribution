package utils

import (
	"fmt"
	"sync"
	"time"
)

// 雪花算法（Snowflake）生成分布式唯一 ID。
// ID 结构（64位，最高位符号位固定为 0，共63位有效）：
//
//	1位     41位           5位       5位     12位
//
// | 0 | 时间戳（毫秒） | workerID | dataID | 序列号 |
const (
	workerBits = 5  // workerID 占用位数
	dataBits   = 5  // dataID 占用位数
	seqBis     = 12 // 序列号占用位数，每毫秒最多生成 4096 个 ID

	maxWorker = -1 ^ (-1 << workerBits) // workerID 最大值：31
	maxData   = -1 ^ (-1 << dataBits)   // dataID 最大值：31
	maxSeq    = -1 ^ (-1 << seqBis)     // 序列号最大值：4095

	// 各字段在 ID 中的左移位数
	timeShift   = workerBits + dataBits + seqBis // 时间戳左移 22 位
	workerShift = dataBits + seqBis              // workerID 左移 17 位
	dataShift   = seqBis                         // dataID 左移 12 位

	twepoch = 1609000000000 // 自定义起始纪元（毫秒时间戳），减小时间戳部分的数值，延长可用年限
)

// Snowflake 雪花 ID 生成器
type Snowflake struct {
	mu       sync.Mutex // 保证并发安全
	lastTs   int64      // 上次生成 ID 的时间戳（毫秒）
	workerID int64      // 工作节点 ID，范围 [0, 31]
	dataID   int64      // 数据中心 ID，范围 [0, 31]
	seq      int64      // 当前毫秒内的序列号，范围 [0, 4095]
}

// NewSnowflake 创建雪花 ID 生成器，workerID 和 dataID 超出范围时直接 panic。
// 启动时等待 1ms，确保重启后生成的时间戳不与上次运行的历史 ID 重叠。
func NewSnowflake(workerID, dataID int64) *Snowflake {
	if workerID < 0 || workerID > maxWorker {
		panic("worker id invalid")
	}
	if dataID < 0 || dataID > maxData {
		panic("data id invalid")
	}
	// 等待 1ms，避免程序重启后 lastTs 归零导致同一毫秒内生成重复 ID
	time.Sleep(time.Millisecond)
	return &Snowflake{
		workerID: workerID,
		dataID:   dataID,
	}
}

// NextID 生成下一个唯一 ID。
// 时钟回拨时返回 error；同一毫秒内序列号耗尽时，阻塞等待到下一毫秒再生成。
func (s *Snowflake) NextID() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ts := time.Now().UnixMilli()

	// 时钟回拨：拒绝生成，避免产生重复 ID
	if ts < s.lastTs {
		return 0, fmt.Errorf("clock moved backwards, refusing to generate id for %d ms", s.lastTs-ts)
	}

	if ts == s.lastTs {
		// 同一毫秒内：序列号自增
		s.seq = (s.seq + 1) & maxSeq
		if s.seq == 0 {
			// 序列号已耗尽（超过 4095），等待推进到下一毫秒
			for ts <= s.lastTs {
				ts = time.Now().UnixMilli()
			}
		}
	} else {
		// 新的毫秒：序列号重置
		s.seq = 0
	}
	s.lastTs = ts

	// 按位拼装：时间戳 | workerID | dataID | 序列号
	return ((ts - twepoch) << timeShift) | (s.dataID << dataShift) | (s.workerID << workerShift) | s.seq, nil
}

// 包级默认生成器（workerID=1, dataID=1），适用于单机场景
var sf = NewSnowflake(1, 1)

// GenID 使用默认生成器生成唯一 ID
func GenID() (int64, error) {
	return sf.NextID()
}
