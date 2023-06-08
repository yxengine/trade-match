package order_book

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"
)

type Serializer interface {
	Serialize(interface{}) ([]byte, error)
	Deserialize([]byte, interface{}) error
}

type JSONSerializer struct{}

func (s JSONSerializer) Serialize(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

func (s JSONSerializer) Deserialize(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

type OrderType int

const (
	LimitOrder  OrderType = iota // 限价单
	MarketOrder                  // 市价单
)

type Order struct {
	ID         int
	Type       OrderType
	Price      float64
	Amount     float64
	Priority   int
	CreateTime time.Time
	ProductID  int
}

type OrderBook struct {
	BuyOrders     map[int][]Order
	SellOrders    map[int][]Order
	PriceTolerance float64
	mutex         sync.Mutex
	Serializer    Serializer
}

type OrderQueue struct {
	Orders []Order
}

func NewOrderBook(serializer Serializer, priceTolerance float64) *OrderBook {
	return &OrderBook{
		BuyOrders:     make(map[int][]Order),
		SellOrders:    make(map[int][]Order),
		PriceTolerance: priceTolerance,
		Serializer:    serializer,
	}
}

func (ob *OrderBook) AddBuyOrder(order Order) {
	ob.mutex.Lock()
	defer ob.mutex.Unlock()

	ob.insertOrder(ob.BuyOrders, order)
}

func (ob *OrderBook) AddSellOrder(order Order) {
	ob.mutex.Lock()
	defer ob.mutex.Unlock()

	ob.insertOrder(ob.SellOrders, order)
}

func (ob *OrderBook) insertOrder(orderMap map[int][]Order, order Order) {
	orders := orderMap[order.ProductID]
	orders = append(orders, order)
	orderMap[order.ProductID] = orders
}

func (ob *OrderBook) CancelBuyOrder(productID, orderID int) {
	ob.mutex.Lock()
	defer ob.mutex.Unlock()

	orders := ob.BuyOrders[productID]
	for i, order := range orders {
		if order.ID == orderID {
			ob.BuyOrders[productID] = append(orders[:i], orders[i+1:]...)
			break
		}
	}
}

func (ob *OrderBook) CancelSellOrder(productID, orderID int) {
	ob.mutex.Lock()
	defer ob.mutex.Unlock()

	orders := ob.SellOrders[productID]
	for i, order := range orders {
		if order.ID == orderID {
			ob.SellOrders[productID] = append(orders[:i], orders[i+1:]...)
			break
		}
	}
}

func (ob *OrderBook) MatchOrders(productID int) {
	ob.mutex.Lock()
	defer ob.mutex.Unlock()

	tradeChannel := make(chan Order)
	wg := sync.WaitGroup{}

	// 启动撮合协程
	wg.Add(1)
	go ob.processMatchOrders(productID, tradeChannel, &wg)

	// 将买单和卖单发送到撮合协程进行撮合
	buyOrders := ob.BuyOrders[productID]
	sellOrders := ob.SellOrders[productID]
	for _, buyOrder := range buyOrders {
		for _, sellOrder := range sellOrders {
			if ob.shouldMatch(buyOrder, sellOrder) {
				tradeChannel <- Order{
					ID:         buyOrder.ID,
					Type:       buyOrder.Type,
					Price:      buyOrder.Price,
					Amount:     buyOrder.Amount,
					Priority:   buyOrder.Priority,
					CreateTime: buyOrder.CreateTime,
					ProductID:  productID,
				}
				tradeChannel <- Order{
					ID:         sellOrder.ID,
					Type:       sellOrder.Type,
					Price:      sellOrder.Price,
					Amount:     sellOrder.Amount,
					Priority:   sellOrder.Priority,
					CreateTime: sellOrder.CreateTime,
					ProductID:  productID,
				}
			}
		}
	}

	// 关闭撮合通道，等待撮合协程结束
	close(tradeChannel)
	wg.Wait()
}

func (ob *OrderBook) processMatchOrders(productID int, tradeChannel <-chan Order, wg *sync.WaitGroup) {
	defer wg.Done()

	for tradeOrder := range tradeChannel {
		ob.mutex.Lock()

		// 找到对应的买单和卖单
		var buyOrder, sellOrder *Order
		for i := range ob.BuyOrders[productID] {
			if ob.BuyOrders[productID][i].ID == tradeOrder.ID {
				buyOrder = &ob.BuyOrders[productID][i]
				break
			}
		}
		for i := range ob.SellOrders[productID] {
			if ob.SellOrders[productID][i].ID == tradeOrder.ID {
				sellOrder = &ob.SellOrders[productID][i]
				break
			}
		}

		if buyOrder == nil || sellOrder == nil {
			ob.mutex.Unlock()
			continue
		}

		// 处理市价单的情况
		if buyOrder.Type == MarketOrder {
			buyOrder.Price = sellOrder.Price
		} else if sellOrder.Type == MarketOrder {
			sellOrder.Price = buyOrder.Price
		}

		// 计算成交数量
		tradeAmount := sellOrder.Amount
		if buyOrder.Amount < sellOrder.Amount {
			tradeAmount = buyOrder.Amount
		}

		// 输出成交信息
		fmt.Printf("Trade: Buy Order %d and Sell Order %d for Product %d at Price %.2f, Amount %.2f\n",
			buyOrder.ID, sellOrder.ID, tradeOrder.ProductID, tradeOrder.Price, tradeAmount)

		// 更新订单数量
		buyOrder.Amount -= tradeAmount
		sellOrder.Amount -= tradeAmount

		// 移除数量为0的订单
		if buyOrder.Amount == 0 {
			ob.CancelBuyOrder(productID, buyOrder.ID)
		}
		if sellOrder.Amount == 0 {
			ob.CancelSellOrder(productID, sellOrder.ID)
		}

		ob.mutex.Unlock()
	}
}

func (ob *OrderBook) shouldMatch(buyOrder, sellOrder Order) bool {
	if buyOrder.Price >= sellOrder.Price {
		if buyOrder.Type == MarketOrder || sellOrder.Type == MarketOrder {
			return true
		} else {
			return buyOrder.Price >= sellOrder.Price
		}
	}
	return false
}

func (ob *OrderBook) UpdatePrice(productID int, newPrice float64) {
	ob.mutex.Lock()
	defer ob.mutex.Unlock()

	// 移动满足价格范围的订单到一级队列
	ob.moveOrdersToPrimaryQueue(productID)

	// 更新价格
	buyOrders := ob.BuyOrders[productID]
	sellOrders := ob.SellOrders[productID]
	for i := range buyOrders {
		buyOrders[i].Price = newPrice
	}
	for i := range sellOrders {
		sellOrders[i].Price = newPrice
	}
}

func (ob *OrderBook) moveOrdersToPrimaryQueue(productID int) {
	secondaryBuyOrders := ob.BuyOrders[productID]
	secondarySellOrders := ob.SellOrders[productID]
	primaryBuyOrders := make([]Order, 0)
	primarySellOrders := make([]Order, 0)

	for _, buyOrder := range secondaryBuyOrders {
		if math.Abs(buyOrder.Price-ob.getMarketPrice(productID)) <= ob.PriceTolerance {
			primaryBuyOrders = append(primaryBuyOrders, buyOrder)
		}
	}

	for _, sellOrder := range secondarySellOrders {
		if math.Abs(sellOrder.Price-ob.getMarketPrice(productID)) <= ob.PriceTolerance {
			primarySellOrders = append(primarySellOrders, sellOrder)
		}
	}

	ob.BuyOrders[productID] = primaryBuyOrders
	ob.SellOrders[productID] = primarySellOrders
}

func (ob *OrderBook) getMarketPrice(productID int) float64 {
	buyOrders := ob.BuyOrders[productID]
	sellOrders := ob.SellOrders[productID]

	if len(buyOrders) > 0 && len(sellOrders) > 0 {
		return (buyOrders[0].Price + sellOrders[0].Price) / 2
	} else if len(buyOrders) > 0 {
		return buyOrders[0].Price
	} else if len(sellOrders) > 0 {
		return sellOrders[0].Price
	}

	return 0
}