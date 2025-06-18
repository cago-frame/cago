package storage

import (
	"context"
	"errors"
	"strings"

	"github.com/cago-frame/cago/middleware/permission"
)

type Group struct {
	permission.PolicyStorage
}

func NewGroup(storage permission.PolicyStorage) *Group {
	return &Group{
		PolicyStorage: storage,
	}
}

func (g *Group) FindPolicyBySubject(ctx context.Context, sub string) ([]*permission.Policy, error) {
	// 先搜索到用户属于哪个组
	userPolicy, err := g.PolicyStorage.FindPolicyBySubject(ctx, sub)
	if err != nil {
		return nil, err
	}
	// 再搜索组的策略
	list := make([]*permission.Policy, 0)
	for _, v := range userPolicy {
		if strings.HasPrefix(v.Resource, "group:") {
			policies, err := g.PolicyStorage.FindPolicyBySubject(ctx, v.Resource)
			if err != nil {
				return nil, err
			}
			list = append(list, policies...)
		} else {
			list = append(list, v)
		}
	}
	return list, nil
}

func (g *Group) FindPolicyByResource(ctx context.Context, sub, res string) ([]*permission.Policy, error) {
	// 先搜索到用户属于哪个组
	userPolicy, err := g.PolicyStorage.FindPolicyBySubject(ctx, sub)
	if err != nil {
		return nil, err
	}
	// 再搜索组的策略
	list := make([]*permission.Policy, 0)
	for _, v := range userPolicy {
		if strings.HasPrefix(v.Resource, "group:") {
			policies, err := g.PolicyStorage.FindPolicyByResource(ctx, v.Resource, res)
			if err != nil {
				return nil, err
			}
			list = append(list, policies...)
		} else if v.Resource == res {
			list = append(list, v)
		}
	}
	return list, nil
}

func (g *Group) AddGroupPolicy(ctx context.Context, policy *permission.Policy) error {
	policy.Subject = "group:" + policy.Subject
	return g.PolicyStorage.AddPolicy(ctx, policy)
}

func (g *Group) AddPolicy(ctx context.Context, policy *permission.Policy) error {
	if strings.HasPrefix(policy.Subject, "group:") {
		return errors.New("subject can not start with group")
	}
	return g.PolicyStorage.AddPolicy(ctx, policy)
}

// AddUserToGroup 添加用户到组
func (g *Group) AddUserToGroup(ctx context.Context, user, group string) error {
	return g.AddPolicy(ctx, &permission.Policy{
		Subject:  user,
		Resource: "group:" + group,
	})
}
