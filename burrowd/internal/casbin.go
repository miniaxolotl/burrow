package internal

import (
	"context"
	"fmt"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/pckhoi/casbin-pgx-adapter/v3"
)

func NewCasbinEnforcer(ctx context.Context, pg *PostgresClient) (*casbin.Enforcer, error) {
	adapter, err := pgxadapter.NewAdapter(nil, pgxadapter.WithConnectionPool(pg.pool))
	if err != nil {
		return nil, fmt.Errorf("create casbin adapter: %w", err)
	}

	m, err := model.NewModelFromString(`
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = r.sub == p.sub && r.obj == p.func && r.act == p.act
`)
	if err != nil {
		return nil, fmt.Errorf("create casbin model: %w", err)
	}

	enforcer, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, fmt.Errorf("create casbin enforcer: %w", err)
	}

	if err := enforcer.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("load casbin policy: %w", err)
	}

	return enforcer, nil
}

func Enforce(enforcer *casbin.Enforcer, sub, obj, act string) (bool, error) {
	return enforcer.Enforce(sub, obj, act)
}
