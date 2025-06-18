package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/cago-frame/cago/middleware/permission"
)

var _ permission.PolicyStorage = (*Memory)(nil)

type Memory struct {
	sync.Mutex
	index int
	list  []*permission.Policy
}

func NewMemory() *Memory {
	return &Memory{
		list: make([]*permission.Policy, 0),
	}
}

func (m *Memory) FindPolicyBySubject(ctx context.Context, sub string) ([]*permission.Policy, error) {
	list := make([]*permission.Policy, 0)
	for _, v := range m.list {
		if v.Subject == sub {
			list = append(list, v)
		}
	}
	return list, nil
}

func (m *Memory) FindPolicyByResource(ctx context.Context, sub, res string) ([]*permission.Policy, error) {
	list := make([]*permission.Policy, 0)
	for _, v := range m.list {
		if v.Subject == sub && v.Resource == res {
			list = append(list, v)
		}
	}
	return list, nil
}

func (m *Memory) AddPolicy(ctx context.Context, policy *permission.Policy) error {
	m.Lock()
	defer m.Unlock()
	m.index++
	policy.ID = fmt.Sprintf("%d", m.index)
	m.list = append(m.list, policy)
	return nil
}

func (m *Memory) RemovePolicy(ctx context.Context, policy *permission.Policy) error {
	for i, v := range m.list {
		if v.ID == policy.ID {
			m.list = append(m.list[:i], m.list[i+1:]...)
			return nil
		}
	}
	return permission.ErrPolicyNotFound
}

func (m *Memory) UpdatePolicy(ctx context.Context, policy *permission.Policy) error {
	for i, v := range m.list {
		if v.ID == policy.ID {
			m.list[i] = policy
			return nil
		}
	}
	return permission.ErrPolicyNotFound
}
