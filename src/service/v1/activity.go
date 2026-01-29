package service

import (
	"context"

	"github.com/SimonHofman/EasySwapBackend/src/service/svc"
	"github.com/SimonHofman/EasySwapBackend/src/types/v1"
	"github.com/pkg/errors"
)

func GetMultiChainActivity(ctx context.Context, svcCtx *svc.ServerCtx, chainID []int, chainName []string, collectionAddrs []string, tokenID string, userAddrs []string, eventTypes []string, page, pageSize int) (*types.ActivityResp, error) {
	activities, total, err := svcCtx.Dao.QueryMultiChainActivities(ctx, chainName, collectionAddrs, tokenID, userAddrs, eventTypes, page, pageSize)
	if err != nil {
		return nil, errors.Wrap(err, "failed on query multi-chain activity")
	}

	if total == 0 || len(activities) == 0 {
		return &types.ActivityResp{
			Result: nil,
			Count:  0,
		}, nil
	}

	results, err := svcCtx.Dao.QueryMultiChainActivityExternalInfo(ctx, chainID, chainName, activities)
	if err != nil {
		return nil, errors.Wrap(err, "failed on query activity external info")
	}

	return &types.ActivityResp{
		Result: results,
		Count:  total,
	}, nil
}
