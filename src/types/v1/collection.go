package types

import "github.com/shopspring/decimal"

type CollectionFilterparams struct {
	Sort        int    `json:"sort"`
	Status      []int  `json:"status"`
	Markets     []int  `json:"markets"`
	TokenID     string `json:"token_id"`
	UserAddress string `json:"user_address"`
	ChainID     int    `json:"chain_id"`
	Page        int    `json:"page"`
	PageSize    int    `json:"page_size"`
}

type CollectionBidFilterParams struct {
	ChainID  int `json:"chain_id"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

type CollectionBids struct {
	Price   decimal.Decimal `json:"price"`
	Size    int             `json:"size"`
	Total   decimal.Decimal `json:"total"`
	Bidders int             `json:"bidders"`
}

type CollectionBidsResp struct {
	Result interface{} `json:"result"`
	Count  int64       `json:"count"`
}

type HistorySalesPriceInfo struct {
	Price     decimal.Decimal `json:"price"`
	TokenID   string          `json:"token_id"`
	TimeStamp int64           `json:"time_stamp"`
}

type TopTraitFilterParams struct {
	TokenIds []string `json:"token_ids"`
	ChainID  int      `json:"chain_id"`
}

type NFTListingInfoResp struct {
	Result interface{} `json:"result"`
	Count  int64       `json:"count"`
}

type NFTListingInfo struct {
	Name              string      `json:"name"`
	ImageURI          string      `json:"image_uri"`
	VideoType         string      `json:"video_type"`
	VideoURI          string      `json:"video_uri"`
	CollectionAddress string      `json:"collection_address"`
	TokenID           string      `json:"token_id"`
	OwnerAddress      string      `json:"owner_address"`
	Traits            []ItemTrait `json:"traits"`

	ListOrderID    string          `json:"list_order_id"`
	ListTime       int64           `json:"list_time"`
	ListPrice      decimal.Decimal `json:"list_price"`
	ListExpireTime int64           `json:"list_expire_time"`
	ListSalt       string          `json:"list_salt"`
	ListMaker      string          `json:"list_maker"`

	BidOrderID    string          `json:"bid_order_id"`
	BidTime       int64           `json:"bid_time"`
	BidExpireTime int64           `json:"bid_expire_time"`
	BidPrice      decimal.Decimal `json:"bid_price"`
	BidSalt       int64           `json:"bid_salt"`
	BidMaker      string          `json:"bid_maker"`
	BidType       int64           `json:"bid_type"`
	BidSize       int64           `json:"bid_size"`
	BidUnfilled   int64           `json:"bid_unfilled"`

	MarketID int `json:"market_id"`

	LastSellPrice    decimal.Decimal `json:"last_sell_price"`
	OwnerOwnedAmount int64           `json:"owner_owned_amount"`
}

type ItemTrait struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type CollectionRankingInfo struct {
	ImageUri    string          `json:"image_uri"`
	Name        string          `json:"name"`
	Address     string          `json:"address"`
	FloorPrice  string          `json:"floor_price"`
	FloorChange string          `json:"floor_change"`
	SellPrice   string          `json:"sell_price"`
	Volume      decimal.Decimal `json:"volume"`
	ItemNum     int64           `json:"item_num"`
	ItemOwner   string          `json:"item_owner"`
	ItemSold    int64           `json:"item_sold"`
	ListAmount  int64           `json:"list_amount"`
	ChainID     int             `json:"chain_id"`
}

type CollectionRankingResp struct {
	Result interface{} `json:"result"`
}

type CollectionDetail struct {
	ImageUri       string          `json:"image_uri"`
	Name           string          `json:"name"`
	Address        string          `json:"address"`
	ChainId        int             `json:"chain_id"`
	FloorPrice     decimal.Decimal `json:"floor_price"`
	SellPrice      string          `json:"sell_price"`
	VolumeTotal    decimal.Decimal `json:"volume_total"`
	Volume24h      decimal.Decimal `json:"volume_24h"`
	Sold24h        int64           `json:"sold_24h"`
	ListAmount     int64           `json:"list_amount"`
	TotalSupply    int64           `json:"total_supply"`
	OwnerAmount    int64           `json:"owner_amount"`
	RoyaltyFeeRate string          `json:"royalty_fee_rate"`
}

type CollectionDetailResp struct {
	Result interface{} `json:"result"`
}

type CommonResp struct {
	Result interface{} `json:"result"`
}

type RefreshItem struct {
	ChainID        int64  `json:"chain_id"`
	CollectionAddr string `json:"collection_addr"`
	TokenID        string `json:"token_id"`
}

type CollectionListed struct {
	CollectionAddr string `json:"collection_address"`
	Count          int    `json:"count"`
}
