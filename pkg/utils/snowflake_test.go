package utils

import (
	"sync"
	"testing"
	"time"
)

/*
运行全部测试：
go test ./pkg/utils/

运行单个测试（用 -run 指定测试函数名）：
go test ./pkg/utils/ -run TestNextID_ClockRollback

加 -v 可以看详细输出，加 -race 可以开启竞态检测（并发测试推荐加）：
go test ./pkg/utils/ -v -race
*/

// TestNewSnowflake_ValidParams 测试合法参数边界值
func TestNewSnowflake_ValidParams(t *testing.T) {
	cases := [][2]int64{
		{0, 0},
		{maxWorker, maxData},
		{0, maxData},
		{maxWorker, 0},
		{1, 1},
	}
	for _, c := range cases {
		sf := NewSnowflake(c[0], c[1])
		if sf == nil {
			t.Errorf("NewSnowflake(%d, %d) returned nil", c[0], c[1])
		}
	}
}

// TestNewSnowflake_InvalidWorkerID 测试非法 workerID 触发 panic
func TestNewSnowflake_InvalidWorkerID(t *testing.T) {
	cases := []int64{-1, maxWorker + 1, -100, 100}
	for _, id := range cases {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("NewSnowflake(workerID=%d, 0) should panic", id)
				}
			}()
			NewSnowflake(id, 0)
		}()
	}
}

// TestNewSnowflake_InvalidDataID 测试非法 dataID 触发 panic
func TestNewSnowflake_InvalidDataID(t *testing.T) {
	cases := []int64{-1, maxData + 1, -100, 100}
	for _, id := range cases {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("NewSnowflake(0, dataID=%d) should panic", id)
				}
			}()
			NewSnowflake(0, id)
		}()
	}
}

// TestNextID_Positive 测试生成的 ID 为正数
func TestNextID_Positive(t *testing.T) {
	sf := NewSnowflake(1, 1)
	id, err := sf.NextID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive ID, got %d", id)
	}
}

// TestNextID_Unique 顺序生成多个 ID，验证无重复
func TestNextID_Unique(t *testing.T) {
	sf := NewSnowflake(1, 1)
	const n = 10000
	ids := make(map[int64]struct{}, n)
	for i := 0; i < n; i++ {
		id, err := sf.NextID()
		if err != nil {
			t.Fatalf("unexpected error at iteration %d: %v", i, err)
		}
		if _, exists := ids[id]; exists {
			t.Fatalf("duplicate ID at iteration %d: %d", i, id)
		}
		ids[id] = struct{}{}
	}
}

// TestNextID_Monotonic 验证 ID 单调递增
func TestNextID_Monotonic(t *testing.T) {
	sf := NewSnowflake(1, 1)
	prev, _ := sf.NextID()
	for i := 0; i < 10000; i++ {
		curr, err := sf.NextID()
		if err != nil {
			t.Fatalf("unexpected error at iteration %d: %v", i, err)
		}
		if curr <= prev {
			t.Fatalf("ID not monotonically increasing: prev=%d, curr=%d", prev, curr)
		}
		prev = curr
	}
}

// TestNextID_BitStructure 验证 ID 中各字段的位编码正确
func TestNextID_BitStructure(t *testing.T) {
	const wid, did = int64(10), int64(20)
	sf := NewSnowflake(wid, did)

	before := time.Now().UnixMilli()
	id, err := sf.NextID()
	after := time.Now().UnixMilli()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	extractedSeq := id & maxSeq
	extractedData := (id >> dataShift) & maxData
	extractedWorker := (id >> workerShift) & maxWorker
	extractedTs := (id >> timeShift) + twepoch

	if extractedWorker != wid {
		t.Errorf("workerID mismatch: got %d, want %d", extractedWorker, wid)
	}
	if extractedData != did {
		t.Errorf("dataID mismatch: got %d, want %d", extractedData, did)
	}
	if extractedTs < before || extractedTs > after {
		t.Errorf("timestamp %d out of range [%d, %d]", extractedTs, before, after)
	}
	if extractedSeq < 0 || extractedSeq > maxSeq {
		t.Errorf("seq %d out of valid range [0, %d]", extractedSeq, maxSeq)
	}
}

// TestNextID_SameMillisecond 验证同一毫秒内 seq 自增
func TestNextID_SameMillisecond(t *testing.T) {
	sf := NewSnowflake(0, 0)
	now := time.Now().UnixMilli()
	sf.lastTs = now
	sf.seq = 5

	id, err := sf.NextID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	seq := id & maxSeq
	ts := (id >> timeShift) + twepoch

	if ts == now {
		// 仍在同一毫秒，seq 应从 5 自增到 6
		if seq != 6 {
			t.Errorf("same ms: expected seq=6, got %d", seq)
		}
	} else {
		// 时间推进了，seq 应重置为 0
		if seq != 0 {
			t.Errorf("new ms: expected seq=0, got %d", seq)
		}
	}
}

// TestNextID_SeqOverflow 验证 seq 溢出后等待下一毫秒再生成
func TestNextID_SeqOverflow(t *testing.T) {
	sf := NewSnowflake(0, 0)
	now := time.Now().UnixMilli()
	sf.lastTs = now
	sf.seq = maxSeq // 下次自增会溢出为 0，触发等待

	id, err := sf.NextID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	seq := id & maxSeq
	ts := (id >> timeShift) + twepoch

	// seq 溢出后必须推进到下一毫秒，所以 ts > now
	if ts <= now {
		t.Errorf("expected ts > %d after seq overflow, got %d", now, ts)
	}
	if seq != 0 {
		t.Errorf("expected seq=0 after overflow, got %d", seq)
	}
}

// TestNextID_ClockRollback 验证时钟回拨时返回错误
func TestNextID_ClockRollback(t *testing.T) {
	sf := NewSnowflake(1, 1)
	sf.lastTs = time.Now().UnixMilli() + 5000 // 模拟 lastTs 在未来 5 秒

	_, err := sf.NextID()
	if err == nil {
		t.Fatal("expected error on clock rollback, got nil")
	}
}

// TestNextID_ConcurrentUnique 并发生成 ID，验证无重复
func TestNextID_ConcurrentUnique(t *testing.T) {
	sf := NewSnowflake(1, 1)
	const goroutines = 20
	const perGoroutine = 500
	total := goroutines * perGoroutine

	idCh := make(chan int64, total)
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				id, err := sf.NextID()
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				idCh <- id
			}
		}()
	}
	wg.Wait()
	close(idCh)

	seen := make(map[int64]struct{}, total)
	for id := range idCh {
		if _, exists := seen[id]; exists {
			t.Fatalf("duplicate ID in concurrent test: %d", id)
		}
		seen[id] = struct{}{}
	}
}

// TestNextID_DifferentWorkers 不同 workerID 同时生成的 ID 不重复
func TestNextID_DifferentWorkers(t *testing.T) {
	sf1 := NewSnowflake(0, 0)
	sf2 := NewSnowflake(1, 0)
	const n = 1000

	set := make(map[int64]struct{}, n*2)
	for i := 0; i < n; i++ {
		id1, _ := sf1.NextID()
		id2, _ := sf2.NextID()
		if _, exists := set[id1]; exists {
			t.Fatalf("duplicate ID from sf1: %d", id1)
		}
		set[id1] = struct{}{}
		if _, exists := set[id2]; exists {
			t.Fatalf("duplicate ID from sf2: %d", id2)
		}
		set[id2] = struct{}{}
	}
}

// TestGenID 验证包级 GenID 可正常使用
func TestGenID(t *testing.T) {
	id, err := GenID()
	if err != nil {
		t.Fatalf("GenID error: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive ID, got %d", id)
	}
}

// TestGenID_Unique 连续调用 GenID 无重复
func TestGenID_Unique(t *testing.T) {
	const n = 1000
	ids := make(map[int64]struct{}, n)
	for i := 0; i < n; i++ {
		id, err := GenID()
		if err != nil {
			t.Fatalf("GenID error at iteration %d: %v", i, err)
		}
		if _, exists := ids[id]; exists {
			t.Fatalf("duplicate ID at iteration %d: %d", i, id)
		}
		ids[id] = struct{}{}
	}
}
