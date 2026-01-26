package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/SimonHofman/EasySwapBackend/src/types/v1"
	"github.com/SimonHofman/EasySwapBase/stores/gdb/orderbookmodel/multi"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const CacheActivityNumPrefix = "cache:es:activity:count:"

var eventTypesToID = map[string]int{
	"sale":                  multi.Sale,
	"transfer":              multi.Transfer,
	"offer":                 multi.MakeOffer,
	"cancel_offer":          multi.CancelOffer,
	"cancel_list":           multi.CancelListing,
	"list":                  multi.Listing,
	"mint":                  multi.Mint,
	"buy":                   multi.Buy,
	"collection_bid":        multi.CollectionBid,
	"item_bid":              multi.ItemBid,
	"cancel_collection_bid": multi.CancelCollectionBid,
	"cancel_item_bid":       multi.CancelItemBid,
}

var idToEventTypes = map[int]string{
	multi.Sale:                "sale",
	multi.Transfer:            "transfer",
	multi.MakeOffer:           "offer",
	multi.CancelOffer:         "cancel_offer",
	multi.CancelListing:       "cancel_list",
	multi.Listing:             "list",
	multi.Mint:                "mint",
	multi.Buy:                 "buy",
	multi.CollectionBid:       "collection_bid",
	multi.ItemBid:             "item_bid",
	multi.CancelCollectionBid: "cancel_collection_bid",
	multi.CancelItemBid:       "cancel_item_bid",
}

type ActivityCountCache struct {
	Chain             string   `json:"chain"`
	ContractAddresses []string `json:"contract_addresses"`
	TokenId           string   `json:"token_id"`
	UserAddress       string   `json:"user_address"`
	EventTypes        []string `json:"event_types"`
}

type ActivityMultiChainInfo struct {
	multi.Activity
	ChainName string `gorm:"column:chain_name"`
}

func getActivityCountCacheKey(activity *ActivityCountCache) (string, error) {
	uid, err := json.Marshal(activity)
	if err != nil {
		return "", errors.Wrap(err, "failed on marshal activity struct")
	}
	return CacheActivityNumPrefix + string(uid), nil
}

func (d *Dao) QueryMultiChainActivities(ctx context.Context, chainName []string, collectionAddrs []string, tokenID string, userAddrs []string, eventTypes []string, page, pageSize int) ([]ActivityMultiChainInfo, int64, error) {
	var strNums []string

	var total int64
	var activities []ActivityMultiChainInfo

	var events []int
	for _, v := range eventTypes {
		id, ok := eventTypesToID[v]
		if !ok {
			continue
		}
		events = append(events, id)
	}

	sqlHead := "SELECT * FROM ("

	sqlMid := ""
	for _, chain := range chainName {
		if sqlMid != "" {
			sqlMid += " UNION ALL "
		}
		sqlMid += fmt.Sprintf("(select '%s' as chain_name,id,collection_address,token_id,currency_address,activity_type,maker,taker,price,tx_hash,event_time,marketplace_id ", chain)
		sqlMid += fmt.Sprintf("from %s ", multi.ActivityTableName(chain))

		if len(userAddrs) == 1 {
			sqlMid += fmt.Sprintf("where maker = '%s' or taker = '%s'", strings.ToLower(userAddrs[0]), strings.ToLower(userAddrs[0]))
		} else if len(userAddrs) > 1 {
			var userAddrsParam string
			for i, addr := range userAddrs {
				userAddrsParam += fmt.Sprintf(`'%s'`, addr)
				if i < len(userAddrs)-1 {
					userAddrsParam += ", "
				}
			}
			sqlMid += fmt.Sprintf("where maker in (%s) or taker in (%s)", userAddrsParam, userAddrsParam)
		}
		sqlMid += ") "
	}

	sqlTail := ") as combined"
	firstFlag := true

	if len(collectionAddrs) == 1 {
		sqlTail += fmt.Sprintf("WHERE collection_address = '%s' ", collectionAddrs[0])
		firstFlag = false
	} else if len(collectionAddrs) > 1 {
		sqlTail += fmt.Sprintf("WHERE collection_address in ('%s'", collectionAddrs[0])
		for i := 1; i < len(collectionAddrs); i++ {
			sqlTail += fmt.Sprintf(",'%s'", collectionAddrs[i])
		}
		sqlTail += ") "
		firstFlag = false
	}

	if tokenID != "" {
		if firstFlag {
			sqlTail += fmt.Sprintf("WHERE token_id = '%s' ", tokenID)
			firstFlag = false
		} else {
			sqlTail += fmt.Sprintf("and token_id = '%s' ", tokenID)
		}
	}

	if len(events) > 0 {
		if firstFlag {
			sqlTail += fmt.Sprintf("WHERE activity_type in (%d", events[0])
			for i := 1; i < len(events); i++ {
				sqlTail += fmt.Sprintf(",%d", events[i])
			}
			sqlTail += ") "
			firstFlag = false
		} else {
			sqlTail += fmt.Sprintf("and activity_type in (%d", events[0])
			for i := 1; i < len(events); i++ {
				sqlTail += fmt.Sprintf(",%d", events[i])
			}
			sqlTail += ") "
		}
	}

	sqlTail += fmt.Sprintf("ORDER BY combined.event_time DESC, combined.id DESC limit %d offset %d", pageSize, pageSize*(page-1))

	sql := sqlHead + sqlMid + sqlTail

	if err := d.DB.Raw(sql).Scan(&activities).Error; err != nil {
		return nil, 0, errors.Wrap(err, "failed on query activity")
	}

	sqlCnt := "SELECT COUNT(*) FROM (" + sqlMid + sqlTail

	cacheKey, err := getActivityCountCacheKey(&ActivityCountCache{
		Chain:             "MultiChain",
		ContractAddresses: collectionAddrs,
		TokenId:           tokenID,
		UserAddress:       strings.ToLower(strings.Join(userAddrs, ",")),
		EventTypes:        eventTypes,
	})

	if err != nil {
		return nil, 0, errors.Wrap(err, "failed on get activity number cache key")
	}

	strNum, err := d.KvStore.Get(cacheKey)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed on get activity number from cache")
	}
	strNums = append(strNums, strNum)

	if strNum != "" {
		total, _ = strconv.ParseInt(strNum, 10, 64)
	} else {
		if err := d.DB.Raw(sqlCnt).Scan(&total).Error; err != nil {
			return nil, 0, errors.Wrap(err, "failed on count activity")
		}

		if err := d.KvStore.Setex(cacheKey, strconv.FormatInt(total, 10), 30); err != nil {
			return nil, 0, errors.Wrap(err, "failed on cache activities number")
		}
	}

	return activities, total, nil
}

func (d *Dao) QueryMultiChainActivityExternalInfo(ctx context.Context, chainID []int, chainName []string, activities []ActivityMultiChainInfo) ([]types.ActivityInfo, error) {
	var userAddrs [][]string
	var items [][]string
	var collectionAddrs [][]string
	for _, activity := range activities {
		userAddrs = append(userAddrs,
			[]string{activity.Maker, activity.ChainName},
			[]string{activity.Taker, activity.ChainName})
		items = append(items,
			[]string{activity.CollectionAddress, activity.TokenId, activity.ChainName})
		collectionAddrs = append(collectionAddrs,
			[]string{activity.CollectionAddress, activity.ChainName})
	}

	userAddrs = removeRepeatedElementArr(userAddrs)
	collectionAddrs = removeRepeatedElementArr(collectionAddrs)
	items = removeRepeatedElementArr(items)

	var itemQuery []clause.Expr
	for _, item := range items {
		itemQuery = append(itemQuery, gorm.Expr("(?, ?)", item[0], item[1]))
	}

	collections := make(map[string]multi.Collection)
	itemInfos := make(map[string]multi.Item)
	itemExternals := make(map[string]multi.ItemExternal)

	var wg sync.WaitGroup
	var queryErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		var newItems []multi.Item
		var newItem multi.Item

		for i := 0; i < len(itemQuery); i++ {
			itemDb := d.DB.WithContext(ctx).
				Table(multi.ItemTableName(items[i][2])).
				Select("collection_address, token_id, name").
				Where("(collection_address, token_id) = ?", itemQuery[i])
			if err := itemDb.Scan(&newItem).Error; err != nil {
				queryErr = errors.Wrap(err, "failed on query items info")
				return
			}

			newItems = append(newItems, newItem)
		}

		for _, item := range newItems {
			itemInfos[strings.ToLower(item.CollectionAddress+item.TokenId)] = item
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		var newItems []multi.ItemExternal
		var newItem multi.ItemExternal

		for i := 0; i < len(itemQuery); i++ {
			itemDb := d.DB.WithContext(ctx).
				Table(multi.ItemExternalTableName(items[i][2])).
				Select("collection_address, token_id, is_uploaded_oss, image_uri, oss_uri").
				Where("(collection_address, token_id) = ?", itemQuery[i])
			if err := itemDb.Scan(&newItem).Error; err != nil {
				queryErr = errors.Wrap(err, "failed on query items info")
				return
			}

			newItems = append(newItems, newItem)
		}

		for _, item := range newItems {
			itemExternals[strings.ToLower(item.CollectionAddress+item.TokenId)] = item
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		var colls []multi.Collection
		var coll multi.Collection

		for i := 0; i < len(collectionAddrs); i++ {
			if err := d.DB.WithContext(ctx).
				Table(multi.CollectionTableName(collectionAddrs[i][1])).
				Select("id, name, address, image_uri").
				Where("address = ?", collectionAddrs[i][0]).
				Scan(&coll).Error; err != nil {
				queryErr = errors.Wrap(err, "failed on query collections info")
				return
			}
			colls = append(colls, coll)
		}

		for _, c := range colls {
			collections[strings.ToLower(c.Address)] = c
		}
	}()
	wg.Wait()

	if queryErr != nil {
		return nil, errors.Wrap(queryErr, "failed on query activity external info")
	}

	chainnameTochainid := make(map[string]int)
	for i, name := range chainName {
		chainnameTochainid[name] = chainID[i]
	}

	var results []types.ActivityInfo
	for _, act := range activities {
		activity := types.ActivityInfo{
			EventType:         "unknown",
			EventTime:         act.EventTime,
			CollectionAddress: act.CollectionAddress,
			TokenID:           act.TokenId,
			Currency:          act.CurrencyAddress,
			Price:             act.Price,
			Maker:             act.Maker,
			Taker:             act.Taker,
			TxHash:            act.TxHash,
			MarketplaceID:     act.MarketplaceID,
			ChainID:           chainnameTochainid[act.ChainName],
		}

		if act.ActivityType == multi.Listing {
			activity.TxHash = ""
		}

		eventType, ok := idToEventTypes[act.ActivityType]
		if ok {
			activity.EventType = eventType
		}

		item, ok := itemInfos[strings.ToLower(act.CollectionAddress+act.TokenId)]
		if ok {
			activity.ItemName = item.Name
		}

		if activity.ItemName == "" {
			activity.ItemName = fmt.Sprintf("#%s", act.TokenId)
		}

		itemExternal, ok := itemExternals[strings.ToLower(act.CollectionAddress+act.TokenId)]
		if ok {
			imageUri := itemExternal.ImageUri
			if itemExternal.IsUploadedOss {
				imageUri = itemExternal.OssUri
			}
			activity.ImageURI = imageUri
		}

		collection, ok := collections[strings.ToLower(act.CollectionAddress)]
		if ok {
			activity.CollectionName = collection.Name
			activity.CollectionImageURI = collection.ImageUri
		}

		results = append(results, activity)
	}

	return results, nil
}

func removeRepeatedElement(arr []string) (newArr []string) {
	newArr = make([]string, 0)
	for i := 0; i < len(arr); i++ {
		repeat := false
		for j := i + 1; j < len(arr); j++ {
			if arr[i] == arr[j] {
				repeat = true
				break
			}
		}
		if !repeat && arr[i] != "" {
			newArr = append(newArr, arr[i])
		}
	}
	return
}

func removeRepeatedElementArr(arr [][]string) [][]string {
	filteredTokenIds := make([][]string, 0)
	seen := make(map[string]bool)

	for _, pair := range arr {
		if len(pair) == 2 {
			key := pair[0] + "," + pair[1]

			if _, exists := seen[key]; !exists {
				filteredTokenIds = append(filteredTokenIds, pair)
				seen[key] = true
			}
		} else if len(pair) == 3 {
			key := pair[0] + "," + pair[1] + "," + pair[2]

			if _, exists := seen[key]; !exists {
				filteredTokenIds = append(filteredTokenIds, pair)
				seen[key] = true
			}
		}
	}

	return filteredTokenIds
}
