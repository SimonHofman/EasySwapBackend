package dao

import (
	"fmt"
	"strings"
	"time"

	"github.com/SimonHofman/EasySwapBase/stores/gdb/orderbookmodel/multi"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

type CollectionTrade struct {
	ContractAddress string          `json:"contract_address"`
	ItemCount       int64           `json:"item_count"`
	Volume          decimal.Decimal `json:"volume"`
	VolumeChange    int             `json:"volume_change"`
	PreFloorPrice   decimal.Decimal `json:"pre_floor_price"`
	FloorChange     int             `json:"floor_change"`
}

func GenRankingKey(project, chain string, period int) string {
	return fmt.Sprintf("cache:%s:%s:ranking:volume:%d", strings.ToLower(project), strings.ToLower(chain), period)
}

type periodEpochMap map[string]int

var periodToEpoch = periodEpochMap{
	"15m": 3,
	"1h":  12,
	"6h":  72,
	"24h": 288,
	"1d":  288,
	"7d":  2016,
	"30d": 8640,
}

func (d *Dao) GetTradeInfoByCollection(chain, collectionAddr, period string) (*CollectionTrade, error) {
	var tradeCount int64
	var totalVolume decimal.Decimal
	var floorPrice decimal.Decimal

	epoch, ok := periodToEpoch[period]
	if !ok {
		return nil, errors.Errorf("invalid period: %s", period)
	}

	startTime := time.Now().Add(-time.Duration(epoch) * time.Minute)
	endTime := time.Now()

	err := d.DB.WithContext(d.ctx).Table(multi.ActivityTableName(chain)).
		Where("collectioin_address = ? AND activity_type = ? AND event_time >= ? AND event_time <= ?",
			collectionAddr, multi.Sale, startTime, endTime).
		Select("COUNT(*) as trade_count, COALESCE(SUM(price), 0) as total_volume").
		Row().Scan(&tradeCount, &totalVolume)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get trade count and volume")
	}

	err = d.DB.WithContext(d.ctx).Table(multi.ActivityTableName(chain)).
		Where("collection_address = ? AND activity_type = ? AND event_time >= ? AND event_time <= ?",
			collectionAddr, multi.Sale, startTime, endTime).
		Select("COALESCE(MIN(price),0)").
		Row().Scan(&floorPrice)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get floor price")
	}

	prevStartTime := startTime.Add(-time.Duration(epoch) * time.Minute)
	prevEndTime := startTime

	var prevVolume decimal.Decimal
	var prevFloorPrice decimal.Decimal

	err = d.DB.WithContext(d.ctx).Table(multi.ActivityTableName(chain)).
		Where("collection_address = ? AND activity_type = ? AND event_time >= ? AND event_time <= ?",
			collectionAddr, multi.Sale, prevStartTime, prevEndTime).
		Select("COALESCE(SUM(price), 0)").
		Row().Scan(&prevVolume)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get previous volume")
	}

	err = d.DB.WithContext(d.ctx).Table(multi.ActivityTableName(chain)).
		Where("collection_address = ? AND activity_type = ? AND event_time >= ? AND event_time <= ?",
			collectionAddr, multi.Sale, prevStartTime, prevEndTime).
		Select("COALESCE(MIN(price), 0)").
		Row().Scan(&prevFloorPrice)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get previous floor price")
	}

	volumeChange := 0
	floorChange := 0

	if !prevVolume.IsZero() {
		volumeChangeDecimal := totalVolume.Sub(prevVolume).Div(prevVolume).Mul(decimal.NewFromInt(100))
		volumeChange = int(volumeChangeDecimal.IntPart())
	}
	if !prevFloorPrice.IsZero() {
		floorChangeDecimal := floorPrice.Sub(prevFloorPrice).Div(prevFloorPrice).Mul(decimal.NewFromInt(100))
		floorChange = int(floorChangeDecimal.IntPart())
	}

	return &CollectionTrade{
		ContractAddress: collectionAddr,
		ItemCount:       tradeCount,
		Volume:          totalVolume,
		VolumeChange:    volumeChange,
		PreFloorPrice:   prevFloorPrice,
		FloorChange:     floorChange,
	}, nil
}

func (d *Dao) GetCollectionRankingByActivity(chain, period string) ([]*CollectionTrade, error) {
	epoch, ok := periodToEpoch[period]
	if !ok {
		return nil, errors.Errorf("invalid period: %s", period)
	}
	startTime := time.Now().Add(-time.Duration(epoch) * time.Minute)
	endTime := time.Now()

	prevEndTime := startTime
	prevStartTime := startTime.Add(-time.Duration(epoch) * time.Minute)

	type TradeStats struct {
		CollectionAddress string
		ItemCount         int64
		Volume            decimal.Decimal
		FloorPrice        decimal.Decimal
	}

	var currentStats []TradeStats
	err := d.DB.WithContext(d.ctx).Table(multi.ActivityTableName(chain)).
		Select("collection_address, COUNT(*) as item_count, COALESCE(SUM(price), 0) as volume, COALESCE(MIN(price), 0) as floor_price").
		Where("activity_type = ? AND event_time >= ? AND event_time <= ? ", multi.Sale, startTime, endTime).
		Group("collection_address").
		Find(&currentStats).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current stats")
	}

	var prevStats []TradeStats
	err = d.DB.WithContext(d.ctx).Table(multi.ActivityTableName(chain)).
		Select("collection_address, COUNT(*) as item_count, COALESCE(SUM(price), 0) as volume, COALESCE(MIN(price), 0) as floor_price").
		Where("activity_type = ? AND event_time >= ? AND event_time <= ?", multi.Sale, prevStartTime, prevEndTime).
		Group("collection_address").
		Find(&prevStats).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed to get previous stats")
	}

	prevStatsMap := make(map[string]TradeStats)
	for _, stat := range prevStats {
		prevStatsMap[stat.CollectionAddress] = stat
	}

	var result []*CollectionTrade
	for _, curr := range currentStats {
		trade := &CollectionTrade{
			ContractAddress: curr.CollectionAddress,
			ItemCount:       curr.ItemCount,
			Volume:          curr.Volume,
			VolumeChange:    0,
			PreFloorPrice:   decimal.Zero,
			FloorChange:     0,
		}

		if prev, ok := prevStatsMap[curr.CollectionAddress]; ok {
			trade.PreFloorPrice = prev.FloorPrice

			if !prev.Volume.IsZero() {
				volumeChangeDecimal := curr.Volume.Sub(prev.Volume).Div(prev.Volume).Mul(decimal.NewFromInt(100))
				trade.VolumeChange = int(volumeChangeDecimal.IntPart())
			}

			if !prev.FloorPrice.IsZero() {
				floorChangeDecimal := curr.FloorPrice.Sub(prev.FloorPrice).Div(prev.FloorPrice).Mul(decimal.NewFromInt(100))
				trade.FloorChange = int(floorChangeDecimal.IntPart())
			}
		}
		result = append(result, trade)
	}

	return result, nil
}

func (d *Dao) GetCollectionVolume(chain, collectionAddr string) (decimal.Decimal, error) {
	var volume decimal.Decimal
	err := d.DB.WithContext(d.ctx).Table(multi.ActivityTableName(chain)).
		Where("collection_address = ? AND activity_type = ? ", collectionAddr, multi.Sale).
		Select("COALESCE(SUM(price), 0)").
		Row().Scan(&volume)
	if err != nil {
		return decimal.Zero, errors.Wrap(err, "failed to get collection volume")
	}

	return volume, nil
}
