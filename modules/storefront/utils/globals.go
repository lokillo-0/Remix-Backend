package utils

import (
	"sync"

	"github.com/remixfn/xenon/modules/storefront/scraper"
)

var (
	StorefrontData map[string][]scraper.Item
	Mu             sync.Mutex
)
