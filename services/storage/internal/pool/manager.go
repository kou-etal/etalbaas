// Package pool manages per-project PostgreSQL connection pools for tenant DBs.
package pool

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/singleflight"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	defaultMaxConns        = 10
	defaultMinConns        = 2
	defaultMaxConnLifetime = 30 * time.Minute
	evictionInterval       = 5 * time.Minute
	idleTTL                = 30 * time.Minute
)

type entry struct {
	pool     *pgxpool.Pool
	lastUsed time.Time
}

// Manager manages per-project PostgreSQL connection pools.
// Connection strings are discovered by reading CNPG-created K8s Secrets.
type Manager struct {
	mu        sync.Mutex
	pools     map[string]*entry
	k8sClient kubernetes.Interface
	sf        singleflight.Group
	stopCh    chan struct{}
	closeOnce sync.Once
}

// NewManager creates a new pool manager.
func NewManager(k8sClient kubernetes.Interface) *Manager {
	m := &Manager{
		pools:     make(map[string]*entry),
		k8sClient: k8sClient,
		stopCh:    make(chan struct{}),
	}
	go m.evictLoop()
	return m
}

// GetPool returns the connection pool for a project, creating one if needed.
func (m *Manager) GetPool(ctx context.Context, projectID string) (*pgxpool.Pool, error) {
	m.mu.Lock()
	if e, ok := m.pools[projectID]; ok {
		e.lastUsed = time.Now()
		m.mu.Unlock()
		return e.pool, nil
	}
	m.mu.Unlock()

	// singleflight ensures only one pool creation per projectID.
	v, err, _ := m.sf.Do(projectID, func() (interface{}, error) {
		// Double-check after acquiring singleflight.
		m.mu.Lock()
		if e, ok := m.pools[projectID]; ok {
			e.lastUsed = time.Now()
			m.mu.Unlock()
			return e.pool, nil
		}
		m.mu.Unlock()

		pool, err := m.createPool(ctx, projectID)
		if err != nil {
			return nil, err
		}

		m.mu.Lock()
		m.pools[projectID] = &entry{pool: pool, lastUsed: time.Now()}
		m.mu.Unlock()

		return pool, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*pgxpool.Pool), nil
}

// WithRLS executes fn within a transaction with RLS enforcement.
// JWT claims are injected as session variables so that auth.uid() and
// auth.role() return correct values for RLS policy evaluation.
func (m *Manager) WithRLS(ctx context.Context, projectID string, claims json.RawMessage, fn func(pgx.Tx) error) error {
	pool, err := m.GetPool(ctx, projectID)
	if err != nil {
		return err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, "SELECT set_config('request.jwt.claims', $1, true)", string(claims)); err != nil {
		return fmt.Errorf("set jwt claims: %w", err)
	}

	role := "authenticated"
	var parsed struct {
		Role string `json:"role"`
	}
	if json.Unmarshal(claims, &parsed) == nil && parsed.Role != "" {
		role = parsed.Role
	}
	// Role name is validated to prevent SQL injection (only known roles allowed).
	if role != "anon" && role != "authenticated" && role != "service_role" {
		role = "authenticated"
	}
	if _, err := tx.Exec(ctx, "SET LOCAL ROLE "+role); err != nil {
		return fmt.Errorf("set role: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Close closes all connection pools. Safe to call multiple times.
func (m *Manager) Close() {
	m.closeOnce.Do(func() {
		close(m.stopCh)
		m.mu.Lock()
		defer m.mu.Unlock()
		for id, e := range m.pools {
			if e.pool != nil {
				e.pool.Close()
			}
			delete(m.pools, id)
		}
	})
}

func (m *Manager) createPool(ctx context.Context, projectID string) (*pgxpool.Pool, error) {
	namespace := "project-" + projectID
	secretName := "db-app"

	secret, err := m.k8sClient.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get tenant db secret %s/%s: %w", namespace, secretName, err)
	}

	username := string(secret.Data["username"])
	password := string(secret.Data["password"])
	if username == "" || password == "" {
		return nil, fmt.Errorf("tenant db secret %s/%s: missing username or password", namespace, secretName)
	}

	host := fmt.Sprintf("db-pooler-rw.%s.svc", namespace)
	dsn := fmt.Sprintf("postgres://%s:%s@%s:5432/postgres", username, password, host)

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse tenant db config: %w", err)
	}
	cfg.MaxConns = defaultMaxConns
	cfg.MinConns = defaultMinConns
	cfg.MaxConnLifetime = defaultMaxConnLifetime

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect tenant db %s: %w", projectID, err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping tenant db %s: %w", projectID, err)
	}

	return pool, nil
}

func (m *Manager) evictLoop() {
	ticker := time.NewTicker(evictionInterval)
	defer ticker.Stop()
	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.evictIdle()
		}
	}
}

func (m *Manager) evictIdle() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for id, e := range m.pools {
		if now.Sub(e.lastUsed) > idleTTL {
			if e.pool != nil {
				e.pool.Close()
			}
			delete(m.pools, id)
		}
	}
}
