package util

import "time"

const maxBackoff = 10

type RateLimiter struct {
	ticker     *time.Ticker
	errorCount int
	baseRate   time.Duration
}

func NewRateLimiter(baseRate time.Duration) RateLimiter {
	rl := RateLimiter{}
	rl.baseRate = baseRate
	rl.ticker = time.NewTicker(rl.baseRate)

	return rl
}

func (rl *RateLimiter) Tick() {

	if rl.ticker != nil {
		<-rl.ticker.C
	}
}

func (rl *RateLimiter) Close() {
	if rl.ticker != nil {
		rl.ticker.Stop()
	}
}

func (rl *RateLimiter) UpdateRate(error bool) {

	update := false
	if error {

		if rl.errorCount < maxBackoff {
			rl.errorCount++
			update = true
		}
	} else if rl.errorCount > 0 {
		rl.errorCount--
		update = true
	}

	if update {
		rl.ticker.Reset(rl.baseRate * time.Duration(rl.errorCount))
	}
}
