package main

import (
	"github.com/markcheno/go-quote"
	"github.com/markcheno/go-trade"
)

func main() {

	//prices, _ := quote.NewQuoteFromYahoo("gld", "2000", "2015", quote.Daily, true)
	prices, _ := quote.NewQuoteFromCSVFile("spy", "spy.csv")
	/*
	   	script := `

	   func LAG(price,period) {
	     lag = make([]float64,len(price))
	     multiplier = 2.0 / (1.0 + period)
	     lag[0] = price[0]
	     for x = 1; x < len(price); x++ {
	   		lag[x] = ( (price[x] - lag[x-1]) * multiplier) + lag[x-1]
	   	}
	     return lag
	   }

	   func ATR(high,low,close,period) {
	     tr  = make([]float64,len(close))
	     tr[0] = high[0] - low[0]
	     for x = 1; x < len(close); x++ {
	       tr1 = high[x] - low[x]
	       tr2 = high[x] - close[x-1]
	       tr3 = close[x-1] - low[x]
	       tr[x] = Max(Max(tr1,tr2),tr3)
	   	}
	     atr = LAG(tr,period)
	     return atr
	   }

	   emafast = LAG(Close,Params[0])
	   emaslow = LAG(Close,Params[1])
	   atr = ATR(High,Low,Close,Params[2])
	   atrmult = Params[3]
	   heat = Params[4]
	   StartCash = 1000000.0
	   StartBar = 25

	   #println(emafast)

	   func run() {

	     # money management
	     risk = atr[Bar] * atrmult
	   	units = Balance[Bar] * heat / risk
	   	SetUnits(units)
	   	#printf("bar=%d, close=%f, atr=%f, balance=%f, risk=%f, units=%f\n",Bar,Close[Bar],atr[Bar],Balance[Bar],risk,Units)

	     if( emafast[Bar] > emaslow[Bar] ) {
	   		BuyOpen()
	   	}
	   	if( emafast[Bar] < emaslow[Bar] ) {
	   		SellOpen()
	   	}
	   }
	   `
	*/
	script2 := `
StartCash = 1000000.0
StartBar = 25
emafast = Ema(Close,20)
emaslow = Ema(Close,50)
SetUnits(500)

func run() {	
  if( emafast[Bar] > emaslow[Bar] ) {
		BuyOpen()
	}
	if( emafast[Bar] < emaslow[Bar] ) {
		SellOpen()
	}
}

	
	`
	params := []float64{15, 150, 20, 5, 0.1}
	strategy := trade.NewStrategy(prices, script2)
	strategy.Backtest(params)
	strategy.Summary()
}
