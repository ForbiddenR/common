// Copyright 2021 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package assets

import (
	"compress/gzip"
	"embed"
	"io"
	"io/fs"
	"time"
)

const (
	gzipSuffix = ".gz"
)

type FileSystem struct {
	embed embed.FS
}

func New(fs embed.FS) FileSystem {
	return FileSystem{fs}
}

// Open implements the fs.FS interface.
func (compressed FileSystem) Open(path string) (fs.File, error) {
	// If we have the file in our embed FS, just return that as it could be a dir.
	var f fs.File
	if f, err := compressed.embed.Open(path); err == nil {
		return f, nil
	}

	f, err := compressed.embed.Open(path + gzipSuffix)
	if err != nil {
		return f, err
	}
	// Read the decompressed content into a buffer.
	gr, err := gzip.NewReader(f)
	if err != nil {
		return f, err
	}
	defer gr.Close()

	c, err := io.ReadAll(gr)
	if err != nil {
		return f, err
	}
	// Wrap everything in our custom File.
	return &File{file: f, content: c}, nil
}

type File struct {
	// The underlying file.
	file fs.File
	// The decrompressed content, needed to return an accurate size.
	content []byte
	// Offset for calls to Read().
	offset int
}

// Stat implements the fs.File interface.
func (f File) Stat() (fs.FileInfo, error) {
	stat, err := f.file.Stat()
	if err != nil {
		return stat, err
	}
	return FileInfo{stat, int64(len(f.content))}, nil
}

// Read implements the fs.File interface.
func (f *File) Read(buf []byte) (int, error) {
	if len(buf) > len(f.content)-f.offset {
		buf = buf[0:len(f.content[f.offset:])]
	}
	n := copy(buf, f.content[f.offset:])
	if n == len(f.content)-f.offset {
		return n, io.EOF
	}
	f.offset += n
	return n, nil
}

// Close implements the fs.File interface.
func (f File) Close() error {
	return f.file.Close()
}

type FileInfo struct {
	fi         fs.FileInfo
	actualSize int64
}

// Name implements the fs.FileInfo interface.
func (fi FileInfo) Name() string {
	name := fi.fi.Name()
	return name[:len(name)-len(gzipSuffix)]
}

// Size implements the fs.FileInfo interface.
func (fi FileInfo) Size() int64 { return fi.actualSize }

// Mode implements the fs.FileInfo interface.
func (fi FileInfo) Mode() fs.FileMode { return fi.fi.Mode() }

// ModTime implements the fs.FileInfo interface.
func (fi FileInfo) ModTime() time.Time { return fi.fi.ModTime() }

// IsDir implements the fs.FileInfo interface.
func (fi FileInfo) IsDir() bool { return fi.fi.IsDir() }

// Sys implements the fs.FileInfo interface.
func (fi FileInfo) Sys() interface{} { return nil }
