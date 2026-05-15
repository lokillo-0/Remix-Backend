package database

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/lzw"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/andr1ww/odin"
	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/utilities"
	bolt "go.etcd.io/bbolt"
)

const (
	CompressionNone = iota
	CompressionGzip
	CompressionZlib
	CompressionFlate
	CompressionLZW
)

type Proxy struct {
	dbs        map[string]*odin.DB
	cache      map[string]CacheEntry
	cacheMutex sync.RWMutex
}

type CacheEntry struct {
	data      []byte
	timestamp time.Time
}

type BucketInfo struct {
	Name     string `json:"name"`
	KeyCount int    `json:"keyCount"`
}

type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type BucketContents struct {
	Database string     `json:"database"`
	Bucket   string     `json:"bucket"`
	Items    []KeyValue `json:"items"`
	Total    int        `json:"total"`
}

type DatabaseSize struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

func NewProxy() *Proxy {
	return &Proxy{
		dbs:   odin.GetAll(),
		cache: make(map[string]CacheEntry),
	}
}

func (p *Proxy) auth(c *gin.Context) {
	ip := c.ClientIP()

	if ip == "127.0.0.1" || ip == "localhost" {
		c.Next()
		return
	}

	if !p.isAdmin(ip) {
		utilities.Authentication.MissingPermission().ApplyC(c.Writer, c)
		return
	}

	c.Next()
}

func (p *Proxy) isAdmin(ip string) bool {
	db, exists := p.dbs["xenon"]
	if !exists {
		return false
	}

	var isAdmin bool
	err := db.Transaction(false, func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("Remix_Admins"))
		if bucket == nil {
			return fmt.Errorf("admin bucket not found")
		}

		return bucket.ForEach(func(k, v []byte) error {
			actualData := p.decompressValue(v)
			if strings.Contains(string(actualData), fmt.Sprintf(`"ip_address":"%s"`, ip)) {
				isAdmin = true
				return fmt.Errorf("found")
			}
			return nil
		})
	})

	if err != nil && err.Error() == "found" {
		return true
	}

	return isAdmin
}

func (p *Proxy) decompressValue(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	cacheKey := fmt.Sprintf("%x", data[:min(16, len(data))])
	p.cacheMutex.RLock()
	if entry, exists := p.cache[cacheKey]; exists {
		if time.Since(entry.timestamp) < 5*time.Minute {
			p.cacheMutex.RUnlock()
			return entry.data
		}
	}
	p.cacheMutex.RUnlock()

	result := p.decompress(data)

	p.cacheMutex.Lock()
	p.cache[cacheKey] = CacheEntry{
		data:      result,
		timestamp: time.Now(),
	}
	if len(p.cache) > 1000 {
		cutoff := time.Now().Add(-5 * time.Minute)
		for k, v := range p.cache {
			if v.timestamp.Before(cutoff) {
				delete(p.cache, k)
			}
		}
	}
	p.cacheMutex.Unlock()

	return result
}

func (p *Proxy) decompress(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	first := data[0]

	if first <= CompressionLZW {
		compressedData := data[1:]

		switch first {
		case CompressionNone:
			return compressedData
		case CompressionGzip:
			if reader, err := gzip.NewReader(bytes.NewReader(compressedData)); err == nil {
				result, _ := io.ReadAll(reader)
				reader.Close()
				return result
			}
		case CompressionZlib:
			if reader, err := zlib.NewReader(bytes.NewReader(compressedData)); err == nil {
				result, _ := io.ReadAll(reader)
				reader.Close()
				return result
			}
		case CompressionFlate:
			reader := flate.NewReader(bytes.NewReader(compressedData))
			result, _ := io.ReadAll(reader)
			reader.Close()
			return result
		case CompressionLZW:
			reader := lzw.NewReader(bytes.NewReader(compressedData), lzw.LSB, 8)
			result, _ := io.ReadAll(reader)
			reader.Close()
			return result
		}
	}

	if len(data) > 0 && (first == 0 || first == 1) {
		if first == 1 {
			if gzReader, err := gzip.NewReader(bytes.NewReader(data[1:])); err == nil {
				result, _ := io.ReadAll(gzReader)
				gzReader.Close()
				return result
			}
		}
		return data[1:]
	}

	if len(data) >= 2 && first == 0x1f && data[1] == 0x8b {
		if gzReader, err := gzip.NewReader(bytes.NewReader(data)); err == nil {
			result, _ := io.ReadAll(gzReader)
			gzReader.Close()
			return result
		}
	}

	return data
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (p *Proxy) getDB(c *gin.Context) (*odin.DB, string, bool) {
	name := c.Param("database")
	if name == "" {
		name = "xenon"
	}
	db, ok := p.dbs[name]
	if !ok {
		utilities.Internal.DataBaseError().WithMessage(fmt.Sprintf("Database '%s' not found", name)).Apply(c.Writer)
	}
	return db, name, ok
}

func (p *Proxy) listDatabases(c *gin.Context) {
	data := make([]gin.H, 0, len(p.dbs))
	for name, db := range p.dbs {
		buckets, err := db.ListBuckets()
		if err != nil {
			utilities.Internal.DataBaseError().WithMessage("Failed to list buckets").Apply(c.Writer)
			return
		}
		data = append(data, gin.H{"name": name, "bucketCount": len(buckets), "buckets": buckets})
	}
	c.JSON(200, data)
}

func (p *Proxy) databaseSizes(c *gin.Context) {
	sizes := make([]DatabaseSize, 0, len(p.dbs))
	for name, db := range p.dbs {
		if size, err := db.GetDiskUsage(); err == nil {
			sizes = append(sizes, DatabaseSize{Name: name, Size: size})
		}
	}
	c.JSON(200, sizes)
}

func (p *Proxy) buckets(c *gin.Context) {
	db, _, ok := p.getDB(c)
	if !ok {
		return
	}

	switch c.Request.Method {
	case "GET":
		var buckets []BucketInfo
		db.Transaction(false, func(tx *bolt.Tx) error {
			return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
				count := 0
				b.ForEach(func(k, v []byte) error {
					count++
					return nil
				})
				buckets = append(buckets, BucketInfo{Name: string(name), KeyCount: count})
				return nil
			})
		})
		c.JSON(200, buckets)

	case "POST":
		var req struct {
			Name string `json:"name" binding:"required"`
		}
		if c.ShouldBindJSON(&req) != nil {
			c.Status(400)
			return
		}
		if err := db.CreateBucket(req.Name); err != nil {
			c.Status(500)
			return
		}
		c.Status(201)

	case "DELETE":
		bucketName := c.Param("bucket")
		if err := db.DeleteBucket(bucketName); err != nil {
			c.Status(500)
			return
		}
		c.Status(200)
	}
}

func (p *Proxy) keys(c *gin.Context) {
	db, dbName, ok := p.getDB(c)
	if !ok {
		return
	}

	bucket := c.Param("bucket")
	key := c.Param("key")

	switch c.Request.Method {
	case "GET":
		if key != "" {
			var value string
			err := db.Transaction(false, func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte(bucket))
				if b == nil {
					return fmt.Errorf("bucket not found")
				}
				val := b.Get([]byte(key))
				if val == nil {
					return fmt.Errorf("key not found")
				}
				actualData := p.decompressValue(val)
				value = string(actualData)
				if pretty, err := json.MarshalIndent(actualData, "", "  "); err == nil {
					value = string(pretty)
				}
				return nil
			})
			if err != nil {
				c.Status(404)
				return
			}
			c.JSON(200, KeyValue{Key: key, Value: value})
		} else {
			search := c.Query("q")
			page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
			offset := (page - 1) * limit

			var items []KeyValue
			total := 0

			db.Transaction(false, func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte(bucket))
				if b == nil {
					return nil
				}
				current := 0
				return b.ForEach(func(k, v []byte) error {
					key := string(k)

					keyMatches := search == "" || strings.Contains(strings.ToLower(key), strings.ToLower(search))

					var actualData []byte
					var value string
					var valueMatches bool

					if !keyMatches && search != "" {
						actualData = p.decompressValue(v)
						value = string(actualData)
						valueMatches = strings.Contains(strings.ToLower(value), strings.ToLower(search))
					} else {
						valueMatches = true
					}

					if keyMatches || valueMatches {
						if current >= offset && len(items) < limit {
							if actualData == nil {
								actualData = p.decompressValue(v)
								value = string(actualData)
							}
							items = append(items, KeyValue{Key: key, Value: value})
						}
						total++
						current++
					}
					return nil
				})
			})
			c.JSON(200, BucketContents{Database: dbName, Bucket: bucket, Items: items, Total: total})
		}

	case "POST":
		var req struct {
			Key   string `json:"key" binding:"required"`
			Value string `json:"value" binding:"required"`
		}
		if c.ShouldBindJSON(&req) != nil {
			c.Status(400)
			return
		}

		var jsonValue interface{}
		if err := json.Unmarshal([]byte(req.Value), &jsonValue); err != nil {
			c.Status(400)
			return
		}
		if err := db.Put(bucket, req.Key, jsonValue); err != nil {
			c.Status(500)
			return
		}
		c.Status(200)

	case "DELETE":
		if err := db.Delete(bucket, key); err != nil {
			c.Status(500)
			return
		}
		c.Status(200)
	}
}

func (p *Proxy) health(c *gin.Context) {
	status := make(map[string]interface{})
	for name, db := range p.dbs {
		err := db.Health()
		s := "healthy"
		if err != nil {
			s = "unhealthy"
		}
		status[name] = gin.H{"status": s}
	}
	c.JSON(200, status)
}

func (p *Proxy) Init(router *gin.Engine) {
	admin := router.Group("/xenon/admin/database", p.auth)
	{
		admin.GET("/databases", p.listDatabases)
		admin.GET("/health", p.health)
		admin.GET("/databases/sizes", p.databaseSizes)
		admin.GET("/:database/buckets", p.buckets)
		admin.POST("/:database/buckets", p.buckets)
		admin.DELETE("/:database/buckets/:bucket", p.buckets)
		admin.GET("/:database/buckets/:bucket/keys", p.keys)
		admin.GET("/:database/buckets/:bucket/keys/:key", p.keys)
		admin.POST("/:database/buckets/:bucket/keys", p.keys)
		admin.GET("/:database/buckets/:bucket/keys/search", p.keys)
		admin.DELETE("/:database/buckets/:bucket/keys/:key", p.keys)
	}
}
