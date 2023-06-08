package order_book

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestOrderBook(t *testing.T) {
	serializer := JSONSerializer{}
	orderBook := NewOrderBook(serializer, 0.05)

	// 添加测试订单
	orderBook.AddBuyOrder(Order{ID: 1, Type: LimitOrder, Price: 10.0, Amount: 5.0, Priority: 3, CreateTime: time.Now(), ProductID: 1})
	orderBook.AddBuyOrder(Order{ID: 2, Type: LimitOrder, Price: 9.5, Amount: 3.0, Priority: 5, CreateTime: time.Now(), ProductID: 1})
	orderBook.AddBuyOrder(Order{ID: 3, Type: LimitOrder, Price: 9.0, Amount: 4.0, Priority: 4, CreateTime: time.Now(), ProductID: 1})

	orderBook.AddSellOrder(Order{ID: 4, Type: LimitOrder, Price: 9.5, Amount: 6.0, Priority: 2, CreateTime: time.Now(), ProductID: 1})
	orderBook.AddSellOrder(Order{ID: 5, Type: LimitOrder, Price: 9.0, Amount: 2.0, Priority: 1, CreateTime: time.Now(), ProductID: 1})

	// 进行撮合操作
	orderBook.MatchOrders(1)

	// 打印撮合后的订单簿状态
	fmt.Println("Final Order Book:")
	fmt.Println("Product 1 Buy Orders:")
	for _, order := range orderBook.BuyOrders[1] {
		fmt.Printf("ID: %d, Type: %v, Price: %.2f, Amount: %.2f, Priority: %d, CreateTime: %v\n",
			order.ID, order.Type, order.Price, order.Amount, order.Priority, order.CreateTime)
	}

	fmt.Println("Product 1 Sell Orders:")
	for _, order := range orderBook.SellOrders[1] {
		fmt.Printf("ID: %d, Type: %v, Price: %.2f, Amount: %.2f, Priority: %d, CreateTime: %v\n",
			order.ID, order.Type, order.Price, order.Amount, order.Priority, order.CreateTime)
	}
}

func BenchmarkMatchOrders(b *testing.B) {
	// 创建撮合模块
	serializer := JSONSerializer{}
	orderBook := NewOrderBook(serializer, 0.05)

	// 添加大量订单
	numOrders := 1000000
	for i := 0; i < numOrders; i++ {
		order := Order{
			ID:         i,
			Type:       LimitOrder,
			Price:      rand.Float64() * 100,
			Amount:     rand.Float64() * 10,
			Priority:   rand.Intn(10),
			CreateTime: time.Now(),
			ProductID:  1,
		}
		if i%2 == 0 {
			orderBook.AddBuyOrder(order)
		} else {
			orderBook.AddSellOrder(order)
		}
	}

	// 重置基准计数器
	b.ResetTimer()

	// 执行撮合操作
	for i := 0; i < b.N; i++ {
		orderBook.MatchOrders(1)
	}
}
