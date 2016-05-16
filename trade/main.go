package main

import (
	"github.com/markcheno/go-quote"
	"github.com/markcheno/go-trade"
)

func main() {

	prices, _ := quote.NewQuoteFromYahoo("spy", "1990", "2015", quote.Daily, true)

	script := `
emafast = Ema(Close,Params[0])
emaslow = Ema(Close,Params[1])
atr = Atr(High,Low,Close,14)
atrmult = 2.5
heat = 0.01
StartCash = 1000000.0
StartBar = 250
Units = 100.0

func run() {

  # money management
  risk = atr[Bar] * atrmult
  Units = Balance[Bar] * heat / risk	
	printf("bar=%d, balance=%f, Equity=%f\n",Bar,Balance[Bar],Equity[Bar])
	
  if( emafast[Bar] > emaslow[Bar] ) {
		BuyOpen()
		CoverOpen()
	}
	if( emafast[Bar] < emaslow[Bar] ) {
		SellOpen()
		ShortOpen()
	}
}
`
	params := []float64{50, 200}

	strategy := trade.NewStrategy(prices, script)
	strategy.Backtest(params)
	strategy.Summary()
}
