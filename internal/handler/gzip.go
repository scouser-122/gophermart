package handler

import (
	"compress/gzip"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
)

type gzipWriter struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

var compressibleTypes = []string{
	"text/html",
	"application/json",
}

var gzipWriterPool = sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(io.Discard)
	},
}

func getGzipWriter(w http.ResponseWriter) *gzipWriter {
	return &gzipWriter{
		w:  w,
		zw: nil,
	}
}

func (c *gzipWriter) Header() http.Header {
	return c.w.Header()
}

func (c *gzipWriter) Write(p []byte) (int, error) {
	if c.shouldCompress() {
		c.zw = gzipWriterPool.Get().(*gzip.Writer)
		c.zw.Reset(c.w)
		return c.zw.Write(p)
	}
	return c.w.Write(p)
}

func (c *gzipWriter) WriteHeader(statusCode int) {
	if c.shouldCompress() {
		c.Header().Set("Content-Encoding", "gzip")
	}
	c.w.WriteHeader(statusCode)
}

func (c *gzipWriter) shouldCompress() bool {
	contentType := c.w.Header().Get("Content-Type")
	for _, ct := range compressibleTypes {
		if strings.HasPrefix(contentType, ct) {
			return true
		}
	}
	return false
}

func (c *gzipWriter) Close() error {
	if c.zw != nil {
		err := c.zw.Close()
		gzipWriterPool.Put(c.zw)
		return err
	}
	return nil
}

type gzipReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

var gzipReaderPool = sync.Pool{
	New: func() interface{} {
		return new(gzip.Reader)
	},
}

func getGzipReader(r io.ReadCloser) (*gzipReader, error) {
	zr := gzipReaderPool.Get().(*gzip.Reader)
	if err := zr.Reset(r); err != nil {
		return nil, err
	}
	return &gzipReader{
		r:  r,
		zr: zr,
	}, nil
}

func (c *gzipReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

func (c *gzipReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	err := c.zr.Close()
	gzipReaderPool.Put(c.zr)
	c.zr = nil
	return err
}

func shouldDecompressRequest(r *http.Request) bool {
	contentEncoding := r.Header.Values("Content-Encoding")
	return slices.Contains(contentEncoding, "gzip")
}

func GzipMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ow := w

		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		if supportsGzip {
			gzWriter := getGzipWriter(w)
			ow = gzWriter
			defer gzWriter.Close()
		}

		if shouldDecompressRequest(r) {
			var err error
			gzReader, err := getGzipReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = gzReader
			defer gzReader.Close()
		}

		h.ServeHTTP(ow, r)
	}
}
