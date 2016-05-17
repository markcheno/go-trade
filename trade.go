package trade

import (
	"fmt"
	"github.com/markcheno/go-quote"
	"github.com/markcheno/go-talib"
	anko_core "github.com/mattn/anko/builtins"
	"github.com/mattn/anko/vm"
	"math"
	"os"
	"time"
)

// Trade - info for a single trade
type Trade struct {
	Symbol     string
	Kind       string
	Units      int
	EntryDate  time.Time
	EntryPrice float64
	ExitDate   time.Time
	ExitPrice  float64
	Profit     float64
}

// Strategy - main struct to hold trading system info
type Strategy struct {
	Price      quote.Quote
	Script     string
	Runtime    time.Duration
	Startbar   int
	Units      float64
	Roundlot   int
	Startcash  float64
	Skidfrac   float64
	Trades     []Trade
	Balance    []float64
	Barcount   int
	bar        int
	maxdd      float64
	peak       float64
	valley     float64
	position   string
	Equity     []float64
	openprofit []float64
	drawdown   []float64
	buystop    []float64
	sellstop   []float64
	shortstop  []float64
	coverstop  []float64
	env        *vm.Env
}

// NewStrategy -
func NewStrategy(p quote.Quote, script string) Strategy {
	s := Strategy{}
	s.Script = script
	s.Price = p
	s.Barcount = len(p.Close)
	s.Equity = make([]float64, s.Barcount)
	s.Balance = make([]float64, s.Barcount)
	s.openprofit = make([]float64, s.Barcount)
	s.drawdown = make([]float64, s.Barcount)
	s.buystop = make([]float64, s.Barcount)
	s.sellstop = make([]float64, s.Barcount)
	s.shortstop = make([]float64, s.Barcount)
	s.coverstop = make([]float64, s.Barcount)
	s.Startcash = 1000000.0
	s.Balance[0] = s.Startcash
	s.Skidfrac = 0.5
	s.Startbar = 0
	s.maxdd = 0.99
	s.Roundlot = 250
	s.position = "flat"
	return s
}

// SetUnits -
func (s *Strategy) SetUnits(units float64) {
	s.Units = units
}

// Backtest -
func (s *Strategy) Backtest(params []float64) float64 {

	starttime := time.Now()
	s.Balance[0] = s.Startcash

	s.env = vm.NewEnv()

	anko_core.Import(s.env)

	//env.Define("printf", fmt.Printf)
	//env.Define("println", fmt.Println)
	s.env.Define("Bar", s.bar)
	s.env.Define("Open", s.Price.Open)
	s.env.Define("High", s.Price.High)
	s.env.Define("Low", s.Price.Low)
	s.env.Define("Close", s.Price.Close)
	s.env.Define("Volume", s.Price.Volume)
	s.env.Define("Date", s.Price.Date)
	s.env.Define("BuyOpen", s.BuyOpen)
	s.env.Define("SellOpen", s.SellOpen)
	s.env.Define("ShortOpen", s.ShortOpen)
	s.env.Define("CoverOpen", s.CoverOpen)
	s.env.Define("Ema", talib.Ema)
	s.env.Define("Atr", talib.Atr)
	s.env.Define("Balance", s.Balance)
	s.env.Define("Equity", s.Equity)
	s.env.Define("StartCash", s.Startcash)
	s.env.Define("StarBar", s.Startbar)
	s.env.Define("Units", s.Units)
	s.env.Define("SetUnits", s.SetUnits)
	s.env.Define("Params", params)
	s.env.Define("Max", math.Max)
	_, err := s.env.Execute(s.Script)
	if err != nil {
		panic(err)
	}

	v, _ := s.env.Get("StartCash")
	s.Startcash = v.Float()
	s.Balance[0] = s.Startcash

	v, _ = s.env.Get("StartBar")
	s.Startbar = int(v.Int())

	for s.bar = 0; s.bar < s.Barcount-1; s.bar++ {
		s.Evaluate()
		if s.bar >= s.Startbar {

			s.env.Set("Bar", s.bar)
			_, err := s.env.Execute("run()")
			if err != nil {
				fmt.Println(err.Error()[9:])
				os.Exit(0)
			}

		}
	}

	//s.ExitTrade(s.Price.Date[s.Barcount-1], s.Price.Close[s.Barcount-1])
	s.ClosePosition()
	s.Runtime = time.Since(starttime)
	return s.Bliss()
}

// Evaluate -
func (s *Strategy) Evaluate() {

	bar := s.bar
	if s.bar > 0 {
		s.Balance[s.bar] = s.Balance[s.bar-1]
	}

	// check if long protective stop was hit
	if s.position == "long" && s.Price.Low[bar] < s.sellstop[bar] {
		bestprice := math.Min(s.Price.Open[bar], s.sellstop[bar])
		fillprice := s.skidfunction(bestprice, s.Price.Low[bar])
		s.ExitTrade(s.Price.Date[bar], fillprice)
	}

	// check if short protective stop was hit
	if s.position == "short" && s.coverstop[bar] > 0 && s.Price.High[bar] > s.coverstop[bar] {
		bestprice := math.Max(s.Price.Open[bar], s.coverstop[bar])
		fillprice := s.skidfunction(bestprice, s.Price.High[bar])
		s.ExitTrade(s.Price.Date[bar], fillprice)
	}

	// check if entry long stop order was hit
	if s.position == "flat" && s.buystop[bar] > 0 && s.Price.High[bar] > s.buystop[bar] {
		bestprice := math.Max(s.Price.Open[bar], s.buystop[bar])
		bestprice = math.Max(bestprice, s.Price.Low[bar])
		fillprice := s.skidfunction(s.Price.High[bar], bestprice)
		s.EnterTrade("long", s.Price.Symbol, s.Units, s.Price.Date[bar], fillprice)
	}

	// check if entry short stop order was hit
	if s.position == "flat" && s.Price.Low[bar] < s.shortstop[bar] {
		bestprice := math.Min(s.Price.Open[bar], s.shortstop[bar])
		bestprice = math.Min(bestprice, s.Price.High[bar])
		fillprice := s.skidfunction(s.Price.Low[bar], bestprice)
		s.EnterTrade("short", s.Price.Symbol, s.Units, s.Price.Date[bar], fillprice)
	}

	// calculate open profit and equity
	if len(s.Trades) > 0 {
		t := s.Trades[len(s.Trades)-1]
		if s.position == "short" {
			s.openprofit[s.bar] = (t.EntryPrice - s.Price.Close[s.bar]) * float64(t.Units)
		} else if s.position == "long" {
			s.openprofit[bar] = (s.Price.Close[bar] - t.EntryPrice) * float64(t.Units)
		}
	}

	s.Equity[s.bar] = s.Balance[s.bar] + s.openprofit[s.bar]

	// calculate drawdown
	if s.bar == 0 {
		s.peak = s.Balance[0]
		s.valley = s.Balance[0]
	}
	if s.Equity[s.bar] > s.peak {
		s.peak = s.Equity[s.bar]
	}
	if s.Equity[s.bar] < s.valley {
		s.valley = s.Equity[s.bar]
	}
	retrace := s.peak - s.valley
	if retrace > 0 {
		s.drawdown[s.bar] = retrace / s.peak
	} else if s.bar > 0 {
		s.drawdown[s.bar] = s.drawdown[s.bar-1]
	}

}

func (s *Strategy) skidfunction(price1, price2 float64) float64 {
	return price1 + s.Skidfrac*(price2-price1)
}

// ExitTrade -
func (s *Strategy) ExitTrade(date time.Time, exitprice float64) {
	profit := 0.0
	if len(s.Trades) == 0 {
		return
	}
	t := &s.Trades[len(s.Trades)-1]
	if s.position == "long" {
		profit = (exitprice - t.EntryPrice) * float64(t.Units)
	} else if s.position == "short" {
		profit = (t.EntryPrice - exitprice) * float64(t.Units)
	}
	s.Balance[s.bar] = s.Balance[s.bar-1] + profit
	if s.bar < s.Barcount-1 {
		s.openprofit[s.bar+1] = 0
	}
	t.ExitDate = date
	t.ExitPrice = exitprice
	t.Profit = profit
	s.position = "flat"
}

// EnterTrade -
func (s *Strategy) EnterTrade(kind string, symbol string, units float64, entrydate time.Time, entryprice float64) {

	t := Trade{
		Symbol:     symbol,
		Kind:       kind,
		Units:      s.roundunits(units),
		EntryDate:  entrydate,
		EntryPrice: entryprice,
		ExitDate:   entrydate,
		ExitPrice:  0,
		Profit:     0,
	}
	s.Trades = append(s.Trades, t)
	s.position = kind
}

func (s *Strategy) roundunits(units float64) int {
	return s.Roundlot * int((units+0.0001)/float64(s.Roundlot)+0.5)
}

// BuyOpen -
func (s *Strategy) BuyOpen() {
	if s.position == "flat" {
		fillprice := s.skidfunction(s.Price.Open[s.bar+1], s.Price.High[s.bar+1])
		s.EnterTrade("long", s.Price.Symbol, s.Units, s.Price.Date[s.bar+1], fillprice)
	}
}

// SellOpen -
func (s *Strategy) SellOpen() {
	if s.position == "long" {
		fillprice := s.skidfunction(s.Price.Open[s.bar+1], s.Price.Low[s.bar+1])
		s.ExitTrade(s.Price.Date[s.bar+1], fillprice)
	}
}

// ShortOpen -
func (s *Strategy) ShortOpen() {
	if s.position == "flat" {
		fillprice := s.skidfunction(s.Price.Open[s.bar+1], s.Price.Low[s.bar+1])
		s.EnterTrade("short", s.Price.Symbol, s.Units, s.Price.Date[s.bar+1], fillprice)
	}
}

// CoverOpen -
func (s *Strategy) CoverOpen() {
	if s.position == "short" {
		fillprice := s.skidfunction(s.Price.Open[s.bar+1], s.Price.High[s.bar+1])
		s.ExitTrade(s.Price.Date[s.bar+1], fillprice)
	}
}

// BuyStop -
func (s *Strategy) BuyStop(price float64) {
	if s.position == "flat" {
		s.buystop[s.bar+1] = price
	}
}

// SellStop -
func (s *Strategy) SellStop(price float64) {
	if s.position == "long" {
		s.sellstop[s.bar+1] = price
	}
}

// ShortStop -
func (s *Strategy) ShortStop(price float64) {
	if s.position == "flat" {
		s.shortstop[s.bar+1] = price
	}
}

// CoverStop -
func (s *Strategy) CoverStop(price float64) {
	if s.position == "short" {
		s.coverstop[s.bar+1] = price
	}
}

// ClosePosition -
func (s *Strategy) ClosePosition() {
	fillprice := 0.0
	if s.position == "long" {
		fillprice = s.skidfunction(s.Price.Close[s.bar], s.Price.Low[s.bar])
	} else if s.position == "short" {
		fillprice = s.skidfunction(s.Price.Close[s.bar], s.Price.High[s.bar])
	} else if s.position == "flat" {
		return
	}
	s.ExitTrade(s.Price.Date[s.bar], fillprice)
	s.openprofit[s.bar] = 0
	s.Equity[s.bar] = s.Balance[s.bar]
}

// Icagr -
func (s *Strategy) Icagr() float64 {
	icagr := 0.0
	end := len(s.Balance) - 1
	if s.Balance[end] > s.Balance[0] {
		ratio := s.Balance[end] / s.Balance[0]
		daterangeinyears := (s.Price.Date[end].Sub(s.Price.Date[0])).Hours() / 24.0 / 365.25
		icagr = math.Log(ratio) / daterangeinyears
	}
	return icagr
}

// DrawDown -
func (s *Strategy) DrawDown() float64 {
	max := -1.0
	for x := 0; x < len(s.drawdown); x++ {
		if s.drawdown[x] > max {
			max = s.drawdown[x]
		}
	}
	return max
}

// Bliss -
func (s *Strategy) Bliss() float64 {
	if s.DrawDown() > s.maxdd {
		return 0
	}
	if s.DrawDown() > 0 {
		return s.Icagr() / s.DrawDown()
	}
	return 0
}

// Summary -
func (s *Strategy) Summary() {
	s.EquityLog()
	s.TradeLog()
	//s.PriceLog()
	fmt.Printf("Bliss=%f, Icagr=%f, DD=%f\n", s.Bliss(), s.Icagr(), s.DrawDown())
}

// PriceLog -
func (s *Strategy) PriceLog() {
	for n := 0; n < s.Barcount; n++ {
		fmt.Printf("%s OHLC:[ %6.2f %6.2f %6.2f %6.2f ]\n",
			s.Price.Date[n].Format("2006-01-02"),
			s.Price.Open[n],
			s.Price.High[n],
			s.Price.Low[n],
			s.Price.Close[n])
	}
}

// EquityLog -
func (s *Strategy) EquityLog() {
	for n := 0; n < s.Barcount; n++ {
		fmt.Printf("%s %10.2f %10.2f %10.2f\n",
			s.Price.Date[n].Format("2006-01-02"),
			s.Balance[n],
			s.openprofit[n],
			s.Equity[n])
	}
}

// TradeLog -
func (s *Strategy) TradeLog() {
	for _, t := range s.Trades {
		fmt.Printf("%s %s %d %s %7.3f %s %7.3f %6.2f\n",
			t.Symbol,
			t.Kind,
			t.Units,
			t.EntryDate.Format("2006-01-02"),
			t.EntryPrice,
			t.ExitDate.Format("2006-01-02"),
			t.ExitPrice,
			t.Profit)
	}
}
