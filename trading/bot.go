package trading

import (
	"sync"
	"time"

	"github.com/ExchangeUnion/xud-tests/xudclient"
	"github.com/ExchangeUnion/xud-tests/xudrpc"
)

var xud *xudclient.Xud

var openOrders = make(map[string]*openOrder)
var openOrdersLock = sync.RWMutex{}

type placeOrderParameters struct {
	price    float64
	quantity float64
	side     xudrpc.OrderSide
}

type openOrder struct {
	quantityLeft float64

	// What should be placed once the order is filled completely
	toPlace placeOrderParameters
}

// InitTradingBot initializes a new trading bot
func InitTradingBot(wg *sync.WaitGroup, xudclient *xudclient.Xud) {
	xud = xudclient

	wg.Add(1)

	go func() {
		defer wg.Done()

		startXudSubscription()
	}()
}

func startXudSubscription() {
	log.Debug("Subscribing to removed orders")

	err := placeOrders()

	if err == nil && len(openOrders) != 0 {
		// TODO: check if the orders still exist
	}

	err = xud.SubscribeRemovedOrders(orderRemoved)

	if err != nil {
		openOrders = make(map[string]*openOrder)

		log.Error("Lost connection to XUD. Retrying in 5 seconds")
		time.Sleep(5 * time.Second)

		startXudSubscription()
	}
}

func placeOrders() error {
	var wg sync.WaitGroup

	orders := []placeOrderParameters{
		{
			price:    0.86,
			quantity: 0.003,
			side:     xudrpc.OrderSide_BUY,
		},
		{
			price:    0.87,
			quantity: 0.0025,
			side:     xudrpc.OrderSide_BUY,
		},
		{
			price:    0.88,
			quantity: 0.002,
			side:     xudrpc.OrderSide_BUY,
		},
		{
			price:    0.89,
			quantity: 0.0015,
			side:     xudrpc.OrderSide_BUY,
		},
		{
			price:    0.9,
			quantity: 0.001,
			side:     xudrpc.OrderSide_BUY,
		},

		{
			price:    1.1,
			quantity: 0.001,
			side:     xudrpc.OrderSide_SELL,
		},
		{
			price:    1.11,
			quantity: 0.0015,
			side:     xudrpc.OrderSide_SELL,
		},
		{
			price:    1.12,
			quantity: 0.002,
			side:     xudrpc.OrderSide_SELL,
		},
		{
			price:    1.13,
			quantity: 0.0025,
			side:     xudrpc.OrderSide_SELL,
		},
		{
			price:    1.14,
			quantity: 0.003,
			side:     xudrpc.OrderSide_SELL,
		},
	}

	var err error

	for _, order := range orders {
		wg.Add(1)

		go func(order placeOrderParameters) {
			placeErr := placeOrder(order)
			if placeErr != nil {
				err = placeErr
			}

			wg.Done()
		}(order)
	}

	wg.Wait()

	if err != nil {
		log.Warning("Could not place orders: %v", err)
	} else {
		log.Debug("Placed orders")
	}

	return err
}

func placeOrder(params placeOrderParameters) error {
	response, err := xud.PlaceOrderSync(xudrpc.PlaceOrderRequest{
		Price:    params.price,
		Quantity: params.quantity,
		Side:     params.side,
		PairId:   "LTC/BTC",
	})

	if err != nil {
		return err
	}

	var remainingOrder = response.RemainingOrder

	// Place a new order until there is quantity remaining
	if remainingOrder == nil || remainingOrder.Quantity == 0 {
		log.Debug("Nothing left of placed order: placing new one")
		err = placeOrder(params)

		return err
	}

	openOrdersLock.Lock()

	openOrders[remainingOrder.Id] = &openOrder{
		quantityLeft: remainingOrder.Quantity,
		toPlace:      params,
	}

	openOrdersLock.Unlock()

	return err
}

func orderRemoved(removal xudrpc.OrderRemoval) {
	log.Debug("Order removed: %v", removal)

	openOrdersLock.RLock()

	filledOrder := openOrders[removal.OrderId]

	openOrdersLock.RUnlock()

	if filledOrder != nil {
		filledOrder.quantityLeft -= removal.Quantity

		// Check if there is quantity left and place new order if not
		if filledOrder.quantityLeft == 0 {
			log.Debug("Placing new order")
			placeOrder(filledOrder.toPlace)
		}
	}
}
