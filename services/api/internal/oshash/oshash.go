// Package oshash 计算 OpenSubtitles Hash（OSHash / ISDb moviehash）。
//
// 算法：hash = file_size + sum_uint64_le(前 64KB) + sum_uint64_le(后 64KB)
// 参考：https://github.com/opensubtitles/oshash
package oshash

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const chunkSize = 65536

// MinFileSize 可计算 hash 的最小文件大小（128KB）
const MinFileSize = chunkSize * 2

// FileHash 文件 OSHash 结果
type FileHash struct {
	Hash     string // 16 位小写十六进制
	FileSize int64
}

// ComputeFile 从本地视频文件计算 OSHash
func ComputeFile(path string) (*FileHash, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Compute(f)
}

// Compute 从可读文件对象计算 OSHash
func Compute(f *os.File) (*FileHash, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := fi.Size()
	if size < MinFileSize {
		return nil, fmt.Errorf("文件过小，无法计算 OSHash: %d bytes", size)
	}

	buf := make([]byte, chunkSize*2)
	if _, err := f.ReadAt(buf[:chunkSize], 0); err != nil {
		return nil, fmt.Errorf("读取文件头: %w", err)
	}
	if _, err := f.ReadAt(buf[chunkSize:], size-chunkSize); err != nil {
		return nil, fmt.Errorf("读取文件尾: %w", err)
	}

	sum, err := sumUint64LE(buf)
	if err != nil {
		return nil, err
	}
	hash := sum + uint64(size)
	return &FileHash{
		Hash:     fmt.Sprintf("%016x", hash),
		FileSize: size,
	}, nil
}

func sumUint64LE(buf []byte) (uint64, error) {
	if len(buf)%8 != 0 {
		return 0, fmt.Errorf("buffer 长度须为 8 的倍数")
	}
	var hash uint64
	for i := 0; i < len(buf); i += 8 {
		hash += binary.LittleEndian.Uint64(buf[i : i+8])
	}
	return hash, nil
}

// ComputeReader 从 Reader 计算（需已知文件大小，用于测试）
func ComputeReader(r io.ReaderAt, size int64) (*FileHash, error) {
	if size < MinFileSize {
		return nil, fmt.Errorf("文件过小: %d bytes", size)
	}
	buf := make([]byte, chunkSize*2)
	if _, err := r.ReadAt(buf[:chunkSize], 0); err != nil {
		return nil, err
	}
	if _, err := r.ReadAt(buf[chunkSize:], size-chunkSize); err != nil {
		return nil, err
	}
	sum, err := sumUint64LE(buf)
	if err != nil {
		return nil, err
	}
	hash := sum + uint64(size)
	return &FileHash{
		Hash:     fmt.Sprintf("%016x", hash),
		FileSize: size,
	}, nil
}
