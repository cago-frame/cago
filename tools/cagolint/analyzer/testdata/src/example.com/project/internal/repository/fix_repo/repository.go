package fix_repo

import (
	"context"

	"github.com/cago-frame/cago/database/db"
)

func Find(ctx context.Context) {
	_ = db.Default() // want "CAGO6003"
}
