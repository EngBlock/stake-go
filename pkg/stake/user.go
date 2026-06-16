package stake

// User is the authenticated Stake user.
type User struct {
	ID                       string `json:"userId"`
	FirstName                string `json:"firstName"`
	LastName                 string `json:"lastName"`
	EmailAddress             string `json:"emailAddress"`
	MACStatus                string `json:"macStatus"`
	AccountType              string `json:"accountType"`
	RegionIdentifier         string `json:"regionIdentifier"`
	DWAccountNumber          string `json:"dw_AccountNumber,omitempty"`
	CanTradeOnUnsettledFunds *bool  `json:"canTradeOnUnsettledFunds,omitempty"`
	Username                 string `json:"username,omitempty"`
}
