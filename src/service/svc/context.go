package svc

import (
	"context"

	"github.com/SimonHofman/EasySwapBackend/src/config"
	"github.com/SimonHofman/EasySwapBackend/src/dao"
	"github.com/SimonHofman/EasySwapBase/chain/nftchainservice"
	"github.com/SimonHofman/EasySwapBase/logger/xzap"
	"github.com/SimonHofman/EasySwapBase/stores/gdb"
	"github.com/SimonHofman/EasySwapBase/stores/xkv"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/kv"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/gorm"
)

type ServerCtx struct {
	C        *config.Config
	DB       *gorm.DB
	Dao      *dao.Dao
	KvStore  *xkv.Store
	RankKey  string
	NodeSrvs map[int64]*nftchainservice.Service
}

func NewServiceContext(c *config.Config) (*ServerCtx, error) {
	var err error

	_, err = xzap.SetUp(c.Log)
	if err != nil {
		return nil, err
	}

	var kvConf kv.KvConf
	for _, con := range c.Kv.Redis {
		kvConf = append(kvConf, cache.NodeConf{
			RedisConf: redis.RedisConf{
				Host: con.Host,
				Type: con.Type,
				Pass: con.Pass,
			},
			Weight: 1,
		})
	}

	store := xkv.NewStore(kvConf)
	db, err := gdb.NewDB(&c.DB)
	if err != nil {
		return nil, err
	}

	nodeSrvs := make(map[int64]*nftchainservice.Service)
	for _, supported := range c.ChainSupported {
		nodeSrvs[int64(supported.ChainID)], err = nftchainservice.New(context.Background(), supported.Endpoint, supported.Name, supported.ChainID,
			c.MetadataParse.NameTags, c.MetadataParse.ImageTags, c.MetadataParse.AttributesTags,
			c.MetadataParse.TraitNameTags, c.MetadataParse.TraitValueTags)

		if err != nil {
			return nil, errors.Wrap(err, "failed on start onchain sync service")
		}
	}

	dao := dao.New(context.Background(), db, store)
	serverCtx := NewServerCtx(
		WithDB(db),
		WithKv(store),
		WithDao(dao),
	)
	serverCtx.C = c

	serverCtx.NodeSrvs = nodeSrvs

	return serverCtx, nil
}
