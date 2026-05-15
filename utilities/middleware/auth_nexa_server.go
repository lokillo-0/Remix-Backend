package middleware

import (
	"encoding/base64"
	"hash/fnv"
	"strconv"
	"strings"

	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/remixfn/xenon/utilities"
)

type cacheEntry struct {
	expectedHash string
	expiry       time.Time
}

var (
	ServerAuthCache  = make(map[string]cacheEntry)
	ServerCacheMutex sync.RWMutex
	ServerCacheTTL   = 5 * time.Minute
)

func RemixServerAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ah := c.GetHeader("Authorization")
		if ah == "" {
			utilities.Authentication.OAuth.InvalidClient().Apply(c.Writer)
			c.Abort()
			return
		}
		if !strings.Contains(ah, "Basic ") {
			utilities.Authentication.InvalidHeader().Apply(c.Writer)
			c.Abort()
			return
		}

		tokenBase64 := strings.TrimPrefix(ah, "Basic ")
		tokenBytes, err := base64.StdEncoding.DecodeString(tokenBase64)
		if err != nil {
			utilities.Authentication.InvalidHeader().Apply(c.Writer)
			c.Abort()
			return
		}

		token := string(tokenBytes)
		parts := strings.Split(token, ":")
		if len(parts) != 2 {
			utilities.Authentication.OAuth.InvalidClient().Apply(c.Writer)
			c.Abort()
			return
		}

		sessionId := parts[0]
		encodedHash := parts[1]

		decodedHash, err := base64.StdEncoding.DecodeString(encodedHash)
		if err != nil {
			utilities.Authentication.InvalidHeader().Apply(c.Writer)
			c.Abort()
			return
		}

		var expectedHash string
		cacheKey := sessionId

		ServerCacheMutex.RLock()
		entry, found := ServerAuthCache[cacheKey]
		ServerCacheMutex.RUnlock()

		if found && time.Now().Before(entry.expiry) {
			expectedHash = entry.expectedHash
		} else {
			h := fnv.New64a()
			h.Write([]byte(sessionId + utilities.GetConfig().GAMESESSION_SECRET))
			expectedHash = strconv.FormatUint(h.Sum64(), 16)

			ServerCacheMutex.Lock()
			ServerAuthCache[cacheKey] = cacheEntry{
				expectedHash: expectedHash,
				expiry:       time.Now().Add(ServerCacheTTL),
			}
			ServerCacheMutex.Unlock()
		}

		if string(decodedHash) != expectedHash {
			utilities.Authentication.OAuth.InvalidClient().Apply(c.Writer)
			c.Abort()
			return
		}

		c.Next()
	}
}
