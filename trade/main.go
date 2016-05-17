package main

import (
	"github.com/markcheno/go-quote"
	"github.com/markcheno/go-trade"
)

func main() {

	//prices, _ := quote.NewQuoteFromYahoo("gld", "2000", "2015", quote.Daily, true)
	prices, _ := quote.NewQuoteFromCSVFile("spy", "spy.csv")

	script := `
	
func LAG(p,period) {	
  lag = make([]float64,len(p))
  multiplier = 2.0 / (1.0 + period)
  lag[0] = p[0]
  for x = 1; x < len(p); x++ {
		lag[x] = ( (p[x] - lag[x-1]) * multiplier) + lag[x-1]
	}
  return lag
}

func ATR(period) {
  tr  = make([]float64,len(Close))
  tr[0] = High[0] - Low[0]
  for x = 1; x < len(Close); x++ {
    tr1 = High[x] - Low[x]
    tr2 = High[x] - Close[x-1]
    tr3 = Close[x-1] - Low[x]
		tmp = Max(tr1,tr2)
    tr[x] = Max(tmp,tr3)
	}
  atr = LAG(tr,period)
  return atr
}

emafast = LAG(Close,15.0)
emaslow = LAG(Close,150.0)
atr = ATR(20)
atrmult = 5
heat = 0.1
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
	params := []float64{15, 150}

	strategy := trade.NewStrategy(prices, script)
	strategy.Backtest(params)
	strategy.Summary()
}
