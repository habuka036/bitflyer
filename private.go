package bitflyer

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type PrivateAPIClient struct {
	key    string
	secret string
}

type sendChildOrderParams struct {
	ProductCode    string  `json:"product_code"`
	ChildOrderType string  `json:"child_order_type"`
	Side           string  `json:"side"`
	Price          float64 `json:"price"`
	Size           float64 `json:"size"`
	MinuteToExpire int     `json:"minute_to_expire,omitempty"`
	TimeInForce    string  `json:"time_in_force,omitempty"`
}

type childOrderAcceptanceID struct {
	ChildOrderAcceptanceID string `json:"child_order_acceptance_id"`
}

func (p *PrivateAPIClient) CreateOrder(product_code string, child_order_type string, side string, price, size float64, time_in_force string) (string, error) {
	res := childOrderAcceptanceID{}
	if err := p.post("/v1/me/sendchildorder", &sendChildOrderParams{
		ProductCode:    product_code,
		ChildOrderType: child_order_type,
		Side:           side,
		Price:          price,
		Size:           size,
		TimeInForce:    time_in_force,
	}, &res); err != nil {
		return "", err
	}
	return res.ChildOrderAcceptanceID, nil
}

type cancelChildOrderParams struct {
	ProductCode            string `json:"product_code"`
	ChildOrderID           string `json:"child_order_id,omitempty"`
	ChildOrderAcceptanceID string `json:"child_order_acceptance_id,omitempty"`
}

func (p *PrivateAPIClient) CancelOrder(id string) error {
	return p.post("/v1/me/cancelchildorder", &cancelChildOrderParams{
		ProductCode:            "FX_BTC_JPY",
		ChildOrderAcceptanceID: id,
	}, nil)
}

func (p *PrivateAPIClient) CancelAllOrder() error {
	return p.post("/v1/me/cancelallchildorders", &cancelChildOrderParams{
		ProductCode: "FX_BTC_JPY",
	}, nil)
}

type Balance struct {
	CurrencyCode string `json:"currency_code"`
	Amount float64 `json:"amount"`
	Available float64 `json:"available"`
}

func (p *PrivateAPIClient) GetBalance() ([]*Balance, error) {
	balances := []*Balance{}
	err := p.get("/v1/me/getbalance", nil, &balances)
	if err != nil {
		return nil, err
	}
	if len(balances) == 0 {
		return nil, fmt.Errorf("%w; cannot retrieve balances", ErrInvalidResponse)
	}
	return balances, nil
}

type Order struct {
	ID                     int64   `json:"id"`
	ChildOrderID           string  `json:"child_order_id"`
	ProductCode            string  `json:"product_code"`
	Side                   string  `json:"side"`
	ChildOrderType         string  `json:"child_order_type"`
	Price                  float64 `json:"price"`
	AveragePrice           float64 `json:"average_price"`
	Size                   float64 `json:"size"`
	ChildOrderState        string  `json:"child_order_state"`
	ExpireDate             string  `json:"expire_date"`
	ChildOrderDate         string  `json:"child_order_date"`
	ChildOrderAcceptanceID string  `json:"child_order_acceptance_id"`
	OutstandingSize        float64 `json:"outstanding_size"`
	CancelSize             float64 `json:"cancel_size"`
	ExecutedSize           float64 `json:"executed_size"`
	TotalCommission        float64 `json:"total_commission"`
}

func (p *PrivateAPIClient) GetOrder(id string, product_code string) (*Order, error) {
	orders := []*Order{}
	err := p.get("/v1/me/getchildorders", map[string]string{"product_code": product_code, "child_order_acceptance_id": id}, &orders)
	if err != nil {
		return nil, err
	}

	if len(orders) == 0 {
		return nil, fmt.Errorf("%w; order is not gound", ErrInvalidResponse)
	}
	return orders[0], nil
}

func (p *PrivateAPIClient) GetActiveOrders(product_code string) ([]*Order, error) {
	orders := []*Order{}
	err := p.get("/v1/me/getchildorders", map[string]string{"product_code": product_code, "child_order_state": "ACTIVE"}, &orders)
	if err != nil {
		return nil, err
	}

	if len(orders) == 0 {
		return nil, fmt.Errorf("%w; order is not gound", ErrInvalidResponse)
	}
	return orders, nil
}

func (p *PrivateAPIClient) GetOrders(product_code string) ([]*Order, error) {
	orders := []*Order{}
	err := p.get("/v1/me/getchildorders", map[string]string{"product_code": product_code}, &orders)
	if err != nil {
		return nil, err
	}

	if len(orders) == 0 {
		return nil, fmt.Errorf("%w; order is not gound", ErrInvalidResponse)
	}
	return orders, nil
}

type Position struct {
	Side  string  `json:"side"`
	Price float64 `json:"price"`
	Size  float64 `json:"size"`
}

func (p *PrivateAPIClient) GetPositions() ([]*Position, error) {
	ps := []*Position{}
	err := p.get("/v1/me/getpositions", map[string]string{"product_code": "FX_BTC_JPY"}, &ps)
	if err != nil {
		return nil, err
	}
	return ps, nil
}

type Collateral struct {
	Collateral      float64 `json:"collateral"`
	OpenPositionPnl float64 `json:"open_position_pnl"`
	KeepRate        float64 `json:"keep_rate"`
}

func (p *PrivateAPIClient) GetCollateral() (*Collateral, error) {
	c := &Collateral{}
	err := p.get("/v1/me/getcollateral", nil, c)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (p *PrivateAPIClient) get(path string, query map[string]string, response interface{}) error {
	client := http.Client{}
	q := []string{}
	for k, v := range query {
		q = append(q, k+"="+v)
	}
	req, err := http.NewRequest("GET", endpoint+path+"?"+strings.Join(q, "&"), nil)
	if err != nil {
		return err
	}

	if err := p.setAuth(req); err != nil {
		return err
	}

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		bytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("%w want: %d, have: %d; %s", ErrInvalidStatusCode, 200, res.StatusCode, err.Error())
		}
		return fmt.Errorf("%w want: %d, have: %d, msg: %s", ErrInvalidStatusCode, 200, res.StatusCode, string(bytes))
	}

	if err := json.NewDecoder(res.Body).Decode(response); err != nil {
		return fmt.Errorf("%w %s", ErrInvalidResponse, err.Error())
	}
	return nil
}

func (p *PrivateAPIClient) post(path string, request interface{}, response interface{}) error {
	client := http.Client{}

	postBody, err := json.Marshal(request)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", endpoint+path, bytes.NewReader(postBody))
	if err != nil {
		return err
	}

	if err := p.setAuth(req); err != nil {
		return err
	}

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		bytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("%w want: %d, have: %d; %s", ErrInvalidStatusCode, 200, res.StatusCode, err.Error())
		}
		return fmt.Errorf("%w want: %d, have: %d; %s", ErrInvalidStatusCode, 200, res.StatusCode, string(bytes))
	}

	if response == nil {
		return nil
	}

	if err := json.NewDecoder(res.Body).Decode(response); err != nil {
		return fmt.Errorf("%w %s", ErrInvalidResponse, err.Error())
	}

	return nil
}

func readBodyCopy(req *http.Request) (string, error) {
	if req.Body == nil {
		return "", nil
	}
	bodyCopy, err := req.GetBody()
	if err != nil {
		return "", err
	}
	defer bodyCopy.Close()
	body, err := ioutil.ReadAll(bodyCopy)
	if err != nil {
		return "", err
	}
	return string(body), err
}

func hmacSHA256(secret string, text string) (string, error) {
	mac := hmac.New(sha256.New, []byte(secret))
	if _, err := mac.Write([]byte(text)); err != nil {
		return "", err
	}
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func (p *PrivateAPIClient) setAuth(req *http.Request) error {
	key := p.key
	now := time.Now()
	body, err := readBodyCopy(req)
	if err != nil {
		return err
	}
	url := req.URL.Path
	if req.URL.RawQuery != "" {
		url += "?" + req.URL.RawQuery
	}
	sign, err := hmacSHA256(p.secret, fmt.Sprintf("%d%s%s%s", now.Unix(), req.Method, url, string(body)))
	if err != nil {
		return err
	}

	req.Header.Add("ACCESS-KEY", key)
	req.Header.Add("ACCESS-TIMESTAMP", fmt.Sprintf("%d", now.Unix()))
	req.Header.Add("ACCESS-SIGN", sign)
	req.Header.Add("Content-Type", "application/json")
	return nil
}
