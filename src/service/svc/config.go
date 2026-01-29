package svc

import (
	"github.com/SimonHofman/EasySwapBackend/src/dao"
	"github.com/SimonHofman/EasySwapBase/evm/erc"
	"github.com/SimonHofman/EasySwapBase/stores/xkv"
	"gorm.io/gorm"
)

type CtxConfig struct {
	db      *gorm.DB
	dao     *dao.Dao
	kvStore *xkv.Store
	Evm     erc.Erc
}

type CtxOption func(conf *CtxConfig)

func NewServerCtx(options ...CtxOption) *ServerCtx {
	c := &CtxConfig{}

	for _, opt := range options {
		opt(c)
	}

	return &ServerCtx{
		DB:      c.db,
		KvStore: c.kvStore,
		Dao:     c.dao,
	}
}

func WithKv(kv *xkv.Store) CtxOption {
	return func(conf *CtxConfig) {
		conf.kvStore = kv
	}
}

func WithDB(db *gorm.DB) CtxOption {
	return func(conf *CtxConfig) {
		conf.db = db
	}
}

func WithDao(dao *dao.Dao) CtxOption {
	return func(conf *CtxConfig) {
		conf.dao = dao
	}
}
