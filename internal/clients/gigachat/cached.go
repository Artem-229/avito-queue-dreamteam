package gigachat

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type CachedEmbedder struct {
	next      *Client
	cacheFile string
	mu        sync.RWMutex
	cache     map[string][]float32
}

func NewCachedEmbedder(realClient *Client, cachePath string) *CachedEmbedder {
	c := &CachedEmbedder{
		next:      realClient,
		cacheFile: cachePath,
		cache:     make(map[string][]float32),
	}
	c.load()
	return c
}

func (c *CachedEmbedder) load() {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(c.cacheFile)
	if err == nil {
		_ = json.Unmarshal(data, &c.cache)
	}
}

func (c *CachedEmbedder) save() error {
	data, err := json.MarshalIndent(c.cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.cacheFile, data, 0644)
}

func (c *CachedEmbedder) CreateEmbedding(ctx context.Context, text string) ([]float32, error) {
	c.mu.RLock()
	emb, exists := c.cache[text]
	c.mu.RUnlock()

	if exists {
		fmt.Printf("[GIGACHAT CACHE HIT]: Вектор взят из файла для текста: %s\n", text)
		return emb, nil
	}

	fmt.Printf("[GIGACHAT API CALL]: Генерация вектора за токены для текста: %s\n", text)
	emb, err := c.next.CreateEmbedding(ctx, text)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.cache[text] = emb
	_ = c.save()
	c.mu.Unlock()

	return emb, nil
}
