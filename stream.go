package gozstd

import (
	"io"
	"sync"
)

// StreamCompress compresses src into dst.
func StreamCompress(dst io.Writer, src io.Reader) error {
	return streamCompressDictLevel(dst, src, nil, DefaultCompressionLevel)
}

// StreamCompressLevel compresses src into dst using the given compressionLevel.
func StreamCompressLevel(dst io.Writer, src io.Reader, compressionLevel int) error {
	return streamCompressDictLevel(dst, src, nil, compressionLevel)
}

// StreamCompressDict compresses src into dst using the given dict cd.
func StreamCompressDict(dst io.Writer, src io.Reader, cd *CDict) error {
	return streamCompressDictLevel(dst, src, cd, 0)
}

func streamCompressDictLevel(dst io.Writer, src io.Reader, cd *CDict, compressionLevel int) error {
	sc := getSCompressor(compressionLevel)
	sc.zw.Reset(dst, cd, compressionLevel)
	_, err := io.CopyBuffer(sc.zw, src, sc.copyBuf)
	if err == nil {
		err = sc.zw.Close()
	}
	putSCompressor(sc)
	return err
}

type sCompressor struct {
	zw               *Writer
	copyBuf          []byte
	compressionLevel int
}

func getSCompressor(compressionLevel int) *sCompressor {
	p := getSCompressorPool(compressionLevel)
	v := p.Get()
	if v == nil {
		return &sCompressor{
			zw:               NewWriterLevel(nil, compressionLevel),
			copyBuf:          make([]byte, 4*1024),
			compressionLevel: compressionLevel,
		}
	}
	return v.(*sCompressor)
}

func putSCompressor(sc *sCompressor) {
	sc.zw.Reset(nil, nil, sc.compressionLevel)
	p := getSCompressorPool(sc.compressionLevel)
	p.Put(sc)
}

func getSCompressorPool(compressionLevel int) *sync.Pool {
	// Use per-level compressor pools, since Writer.Reset is expensive
	// between distinct compression levels.
	sCompressorPoolLock.Lock()
	p := sCompressorPool[compressionLevel]
	if p == nil {
		p = &sync.Pool{}
		sCompressorPool[compressionLevel] = p
	}
	sCompressorPoolLock.Unlock()
	return p
}

var (
	sCompressorPoolLock sync.Mutex
	sCompressorPool     = make(map[int]*sync.Pool)
)

// StreamDecompress decompresses src into dst.
func StreamDecompress(dst io.Writer, src io.Reader) error {
	return StreamDecompressDict(dst, src, nil)
}

// StreamDecompressDict decompresses src into dst using the given dictionary dd.
func StreamDecompressDict(dst io.Writer, src io.Reader, dd *DDict) error {
	sd := getSDecompressor()
	sd.zr.Reset(src, dd)
	_, err := io.CopyBuffer(dst, sd.zr, sd.copyBuf)
	putSDecompressor(sd)
	return err
}

type sDecompressor struct {
	zr      *Reader
	copyBuf []byte
}

func getSDecompressor() *sDecompressor {
	v := sDecompressorPool.Get()
	if v == nil {
		return &sDecompressor{
			zr:      NewReader(nil),
			copyBuf: make([]byte, 4*1024),
		}
	}
	return v.(*sDecompressor)
}

func putSDecompressor(sd *sDecompressor) {
	sd.zr.Reset(nil, nil)
	sDecompressorPool.Put(sd)
}

var sDecompressorPool sync.Pool
