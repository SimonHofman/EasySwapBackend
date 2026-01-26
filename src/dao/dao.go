package dao

import (
	"context"

	"gorm.io/gorm"

	"github.com/SimonHofman/EasySwapBase/stores/xkv"
)

type Dao struct {
	ctx     context.Context
	DB      *gorm.DB
	KvStore *xkv.Store
}

func New(ctx context.Context, db *gorm.DB, kvStore *xkv.Store) *Dao {
	return &Dao{
		ctx:     ctx,
		DB:      db,
		KvStore: kvStore,
	}
}
