package main

import (
	"sync"
	"syscall"
	"time"
)

var (
	rootPath = "/"
	mu       sync.Mutex
)

func startBackgroundRefresh() {
	refreshMetrics()
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for range ticker.C {
			refreshMetrics()
		}
	}()
}

func refreshMetrics() {
	var stat syscall.Statfs_t
	err := syscall.Statfs(rootPath, &stat)
	if err != nil {
		return
	}

	totalBytes := uint64(stat.Blocks) * uint64(stat.Bsize)
	freeBytes := uint64(stat.Bfree) * uint64(stat.Bsize)
	usedBytes := totalBytes - freeBytes
	usedPercent := float64(usedBytes) / float64(totalBytes) * 100

	mu.Lock()
	diskUsedPercent.Set(usedPercent)
	diskFreeBytes.Set(float64(freeBytes))
	mu.Unlock()
}
