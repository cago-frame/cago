package iam

import (
	"context"

	"github.com/cago-frame/cago"
	"github.com/cago-frame/cago/configs"
	"github.com/cago-frame/cago/pkg/iam/audit"
	"github.com/cago-frame/cago/pkg/iam/authn"
)

type Iam struct {
	Authn *authn.Authn
	Audit *audit.Audit
}

var defaultIAM *Iam

type Options struct {
	authnOpts []authn.Option
	auditOpts []audit.Option
}

type Option func(*Options)

func newOptions(opts ...Option) *Options {
	options := &Options{}
	for _, o := range opts {
		o(options)
	}
	return options
}

func WithAuthnOptions(opts ...authn.Option) Option {
	return func(options *Options) {
		options.authnOpts = opts
	}
}

func WithAuditOptions(opts ...audit.Option) Option {
	return func(options *Options) {
		options.auditOpts = opts
	}
}

// IAM IAM组件 集成了认证、鉴权、审计、会话管理模块
func IAM(database authn.Database, opts ...Option) cago.FuncComponent {
	return func(ctx context.Context, cfg *configs.Config) error {
		defaultIAM = New(database, opts...)
		SetDefault(defaultIAM)
		return nil
	}
}

func New(database authn.Database, opts ...Option) *Iam {
	options := newOptions(opts...)
	return &Iam{
		Authn: authn.New(database, options.authnOpts...),
		Audit: audit.NewAudit(options.auditOpts...),
	}
}

func SetDefault(iam *Iam) {
	defaultIAM = iam
	authn.SetDefault(iam.Authn)
	audit.SetDefault(iam.Audit)
}

func Default() *Iam {
	return defaultIAM
}
