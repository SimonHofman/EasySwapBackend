package service

import (
	"context"

	"github.com/SimonHofman/EasySwapBackend/src/service/svc"
	"github.com/SimonHofman/EasySwapBackend/src/types/v1"
	"github.com/SimonHofman/EasySwapBase/stores/gdb/orderbookmodel/multi"
	"github.com/pkg/errors"
)

func GetOrderInfos(ctx context.Context, svcCtx *svc.ServerCtx, chainID int, chain string, userAddr string, collectionAddr string, tokenIds []string) ([]types.ItemBid, error) {
	var items []types.ItemInfo
	for _, tokenID := range tokenIds {
		items = append(items, types.ItemInfo{
			CollectionAddress: collectionAddr,
			TokenID:           tokenID,
		})
	}

	bids, err := svcCtx.Dao.QueryItemsBestBids(ctx, chain, userAddr, items)
	if err != nil {
		return nil, errors.Wrap(err, "failed on query items best bids")
	}

	itemsBestBids := make(map[string]multi.Order)
	for _, bid := range bids {
		order, ok := itemsBestBids[bid.TokenId]
		if !ok {
			itemsBestBids[bid.TokenId] = bid
			continue
		}
		if bid.Price.GreaterThan(order.Price) {
			itemsBestBids[bid.TokenId] = bid
		}
	}

	collectionBids, err := svcCtx.Dao.QueryCollectionTopNBid(ctx, chain, userAddr, collectionAddr, len(tokenIds))
	if err != nil {
		return nil, errors.Wrap(err, "failed on query collection best bids")
	}

	return processBids(tokenIds, itemsBestBids, collectionBids, collectionAddr), nil
}
