package clientpolicy

import (
	dom "suno-wallets/src/domain/clientpolicy"
	"sync"
	"time"
)

// Comentários em pt-BR: cache in-memory simples com TTL por entrada

type cacheItem struct {
	pol *dom.ClientPolicy
	exp time.Time
}

type InMemoryCache struct {
	data map[string]cacheItem
	mu   sync.RWMutex
	ttl  time.Duration
}

func NewInMemoryCache(ttl time.Duration) *InMemoryCache {
	return &InMemoryCache{data: map[string]cacheItem{}, ttl: ttl}
}

func key(tenantID, cpf string) string { return tenantID + "|" + cpf }

func (c *InMemoryCache) Get(tenantID, cpf string) (*dom.ClientPolicy, bool) {
	c.mu.RLock()
	it, ok := c.data[key(tenantID, cpf)]
	c.mu.RUnlock()
	if !ok || time.Now().After(it.exp) {
		return nil, false
	}
	return it.pol, true
}

func (c *InMemoryCache) Set(tenantID, cpf string, pol *dom.ClientPolicy) {
	c.mu.Lock()
	c.data[key(tenantID, cpf)] = cacheItem{pol: pol, exp: time.Now().Add(c.ttl)}
	c.mu.Unlock()
}

func (c *InMemoryCache) Invalidate(tenantID, cpf string) {
	c.mu.Lock()
	delete(c.data, key(tenantID, cpf))
	c.mu.Unlock()
}
