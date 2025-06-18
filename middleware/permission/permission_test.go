package permission_test

import (
	"context"
	"testing"

	"github.com/cago-frame/cago/middleware/permission"
	"github.com/cago-frame/cago/middleware/permission/storage"

	"github.com/stretchr/testify/assert"
)

func TestACL(t *testing.T) {
	ctx := context.Background()
	storage := storage.NewMemory()
	p := permission.NewPermission(storage)
	_ = storage.AddPolicy(ctx, &permission.Policy{
		Subject:  "user1",
		Resource: "/file",
		Actions:  []string{"read"},
		Effect:   permission.Allow,
	})
	err := p.Check(ctx, &permission.Request{
		Subject:  "user1",
		Resource: "/file",
		Action:   "read",
	})
	assert.Nil(t, err)
	err = p.Check(ctx, &permission.Request{
		Subject:  "user1",
		Resource: "/file",
		Action:   "write",
	})
	assert.Equal(t, err, permission.ErrPermissionDenied)
	// 带超级管理员的匹配器
	_ = storage.AddPolicy(ctx, &permission.Policy{
		Subject: "admin",
		Effect:  permission.Allow,
	})
	_ = storage.AddPolicy(ctx, &permission.Policy{
		Subject:  "admin",
		Resource: "/file2",
		Actions:  []string{"write"},
		Effect:   permission.Deny,
		Priority: -1,
	})
	p = permission.NewPermission(storage, permission.WithMatcher(
		permission.NewOrMatcher(
			permission.FuncMatcher(func(ctx context.Context, request *permission.Request, policyStorage permission.PolicyStorageFinder) ([]*permission.Policy, error) {
				policies, err := policyStorage.FindPolicyBySubject(ctx, request.Subject)
				if err != nil {
					return nil, err
				}
				adminPolicies := make([]*permission.Policy, 0)
				for _, policy := range policies {
					if policy.Subject == "admin" &&
						(policy.Resource == "" || policy.Resource == request.Resource) {
						if len(policy.Actions) == 0 || policy.InAction(request.Action) {
							adminPolicies = append(adminPolicies, policy)
						}
					}
				}
				return adminPolicies, nil
			}),
			permission.NewEqualMatcher()),
	))
	err = p.Check(ctx, &permission.Request{
		Subject:  "admin",
		Resource: "/file",
		Action:   "write",
	})
	assert.Nil(t, err)
	err = p.Check(ctx, &permission.Request{
		Subject:  "admin",
		Resource: "/file2",
		Action:   "write",
	})
	assert.Equal(t, err, permission.ErrPermissionDenied)
	err = p.Check(ctx, &permission.Request{
		Subject:  "admin",
		Resource: "/file2",
		Action:   "read",
	})
	assert.Nil(t, err)
}

func TestRBAC(t *testing.T) {
	ctx := context.Background()
	memStorage := storage.NewMemory()
	group := storage.NewGroup(memStorage)
	p := permission.NewPermission(group)
	_ = group.AddGroupPolicy(ctx, &permission.Policy{
		Subject:  "group1",
		Resource: "/file",
		Actions:  []string{"read"},
	})
	_ = group.AddGroupPolicy(ctx, &permission.Policy{
		Subject:  "group2",
		Resource: "/file",
		Actions:  []string{"write"},
	})
	_ = group.AddUserToGroup(ctx, "user1", "group1")
	err := p.Check(ctx, &permission.Request{
		Subject:  "user1",
		Resource: "/file",
		Action:   "read",
	})
	assert.Nil(t, err)
	err = p.Check(ctx, &permission.Request{
		Subject:  "user1",
		Resource: "/file",
		Action:   "write",
	})
	assert.Equal(t, err, permission.ErrPermissionDenied)
}

func TestABAC(t *testing.T) {
	ctx := context.Background()
	memStorage := storage.NewMemory()
	p := permission.NewPermission(memStorage)
	_ = memStorage.AddPolicy(ctx, &permission.Policy{
		Subject: "user:1",
		Effect:  permission.Allow,
	})
	_ = memStorage.AddPolicy(ctx, &permission.Policy{
		Subject: "user:2",
		Effect:  permission.Allow,
	})
	// 资源用户是自己就可以删除
	_ = p.Check(ctx, &permission.Request{
		Subject:  "user:1",
		Resource: "script",
		Action:   "delete",
		Environment: map[string]string{
			"user_id": "1",
		},
	})
	//assert.Nil(t, err)
	// 需要超级管理员才能删除

}
