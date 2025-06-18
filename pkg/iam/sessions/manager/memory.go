package manager

import (
	"context"
	"sync"

	"github.com/cago-frame/cago/pkg/iam/sessions"
	"github.com/cago-frame/cago/pkg/utils"
)

type memorySessionManager struct {
	storage sync.Map
}

func NewMemorySessionManager() sessions.SessionManager {
	return &memorySessionManager{}
}

func (m *memorySessionManager) Start(ctx context.Context) (*sessions.Session, error) {
	return &sessions.Session{
		Metadata: make(map[string]interface{}),
		Values:   make(map[string]interface{}),
	}, nil
}

func (m *memorySessionManager) Get(ctx context.Context, id string) (*sessions.Session, error) {
	session, ok := m.storage.Load(id)
	if !ok {
		return nil, sessions.ErrSessionNotFound
	}
	return session.(*sessions.Session), nil
}

func (m *memorySessionManager) Save(ctx context.Context, session *sessions.Session) error {
	if session.ID == "" {
		session.ID = utils.RandString(32, utils.Mix)
	}
	m.storage.Store(session.ID, session)
	return nil
}

func (m *memorySessionManager) Delete(ctx context.Context, id string) error {
	m.storage.Delete(id)
	return nil
}

func (m *memorySessionManager) Refresh(ctx context.Context, session *sessions.Session) error {
	if err := m.Delete(ctx, session.ID); err != nil {
		return err
	}
	session.ID = ""
	return m.Save(ctx, session)
}
