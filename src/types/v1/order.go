package types

type OrderInfoParam struct {
	ChainID           int      `json:"chain_id"`
	UserAddress       string   `json:"user_address"`
	CollectionAddress string   `json:"collection_address"`
	TokenIds          []string `json:"token_ids"`
}
