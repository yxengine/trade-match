package main

import (
	"fmt"
	"sync"
)

type OrderType int

const (
	LimitOrder  OrderType = iota // 限价单
	MarketOrder                  // 市价单
)

type Order struct {
	ID       int
	Type     OrderType
	Price    float64
	Amount   float64
	Priority int
}

type OrderBook struct {
	BuyOrders  []Order
	SellOrders []Order
}

func (ob *OrderBook) AddBuyOrder(order Order) {
	ob.BuyOrders = append(ob.BuyOrders, order)
}

func (ob *OrderBook) AddSellOrder(order Order) {
	ob.SellOrders = append(ob.SellOrders, order)
}

func (ob *OrderBook) CancelBuyOrder(orderID int) {
	for i, order := range ob.BuyOrders {
		if order.ID == orderID {
			ob.BuyOrders = append(ob.BuyOrders[:i], ob.BuyOrders[i+1:]...)
			break
		}
	}
}

func (ob *OrderBook) CancelSellOrder(orderID int) {
	for i, order := range ob.SellOrders {
		if order.ID == orderID {
			ob.SellOrders = append(ob.SellOrders[:i], ob.SellOrders[i+1:]...)
			break
		}
	}
}

func (ob *OrderBook) MatchOrders() {
	tradeChannel := make(chan Order)
	wg := sync.WaitGroup{}

	// 启动撮合协程
	wg.Add(1)
	go ob.processMatchOrders(tradeChannel, &wg)

	// 将买单和卖单发送到撮合协程进行撮合
	for _, buyOrder := range ob.BuyOrders {
		for _, sellOrder := range ob.SellOrders {
			if ob.shouldMatch(buyOrder, sellOrder) {
				// 发送撮合订单到通道
				tradeChannel <- Order{
					ID:       buyOrder.ID,
					Type:     buyOrder.Type,
					Price:    sellOrder.Price,
					Amount:   sellOrder.Amount,
					Priority: buyOrder.Priority,
				}
			}
		}
	}

	// 关闭通道，表示撮合结束
	close(tradeChannel)

	wg.Wait()
}

func (ob *OrderBook) processMatchOrders(tradeChannel <-chan Order, wg *sync.WaitGroup) {
	defer wg.Done()

	for tradeOrder := range tradeChannel {
		// 找到对应的买单和卖单
		var buyOrder, sellOrder *Order
		for i := range ob.BuyOrders {
			if ob.BuyOrders[i].ID == tradeOrder.ID {
				buyOrder = &ob.BuyOrders[i]
				break
			}
		}
		for i := range ob.SellOrders {
			if ob.SellOrders[i].Price == tradeOrder.Price && ob.SellOrders[i].Amount == tradeOrder.Amount {
				sellOrder = &ob.SellOrders[i]
				break
			}
		}

		if buyOrder == nil || sellOrder == nil {
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
		fmt.Printf("Trade: Buy Order %d and Sell Order %d at Price %.2f, Amount %.2f\n",
			buyOrder.ID, sellOrder.ID, tradeOrder.Price, tradeAmount)

		// 更新订单数量
		buyOrder.Amount -= tradeAmount
		sellOrder.Amount -= tradeAmount

		// 移除数量为0的订单
		if buyOrder.Amount == 0 {
			ob.CancelBuyOrder(buyOrder.ID)
		}
		if sellOrder.Amount == 0 {
			ob.CancelSellOrder(sellOrder.ID)
		}
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

func main() {
	orderBook := OrderBook{}

	// 添加测试用例
	orderBook.AddBuyOrder(Order{ID: 1, Type: LimitOrder, Price: 10.0, Amount: 5.0, Priority: 5})
	orderBook.AddBuyOrder(Order{ID: 2, Type: MarketOrder, Price: 0, Amount: 3.0, Priority: 3})
	orderBook.AddBuyOrder(Order{ID: 3, Type: LimitOrder, Price: 12.0, Amount: 7.0, Priority: 8})

	orderBook.AddSellOrder(Order{ID: 4, Type: LimitOrder, Price: 11.5, Amount: 10.0, Priority: 4})
	orderBook.AddSellOrder(Order{ID: 5, Type: MarketOrder, Price: 0, Amount: 5.0, Priority: 6})

	// 打印订单簿状态
	fmt.Println("Initial Order Book:")
	fmt.Println("Buy Orders:")
	for _, order := range orderBook.BuyOrders {
		fmt.Printf("ID: %d, Type: %v, Price: %.2f, Amount: %.2f, Priority: %d\n", order.ID, order.Type, order.Price, order.Amount, order.Priority)
	}
	fmt.Println("Sell Orders:")
	for _, order := range orderBook.SellOrders {
		fmt.Printf("ID: %d, Type: %v, Price: %.2f, Amount: %.2f, Priority: %d\n", order.ID, order.Type, order.Price, order.Amount, order.Priority)
	}
	fmt.Println()

	// 进行撮合操作
	orderBook.MatchOrders()

	// 打印撮合后的订单簿状态
	fmt.Println("Final Order Book:")
	fmt.Println("Buy Orders:")
	for _, order := range orderBook.BuyOrders {
		fmt.Printf("ID: %d, Type: %v, Price: %.2f, Amount: %.2f, Priority: %d\n", order.ID, order.Type, order.Price, order.Amount, order.Priority)
	}
	fmt.Println("Sell Orders:")
	for _, order := range orderBook.SellOrders {
		fmt.Printf("ID: %d, Type: %v, Price: %.2f, Amount: %.2f, Priority: %d\n", order.ID, order.Type, order.Price, order.Amount, order.Priority)
	}
	fmt.Println()

	// 撤销订单
	orderBook.CancelBuyOrder(2)
	orderBook.CancelSellOrder(5)

	// 打印撤销后的订单簿状态
	fmt.Println("Order Book after Cancellation:")
	fmt.Println("Buy Orders:")
	for _, order := range orderBook.BuyOrders {
		fmt.Printf("ID: %d, Type: %v, Price: %.2f, Amount: %.2f, Priority: %d\n", order.ID, order.Type, order.Price, order.Amount, order.Priority)
	}
	fmt.Println("Sell Orders:")
	for _, order := range orderBook.SellOrders {
		fmt.Printf("ID: %d, Type: %v, Price: %.2f, Amount: %.2f, Priority: %d\n", order.ID, order.Type, order.Price, order.Amount, order.Priority)
	}
}
