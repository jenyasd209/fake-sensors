package generator

import (
	"sync"

	"github.com/gocolly/colly/v2"
)

const source = "https://oceana.org/ocean-fishes/"

func ParseFishNames() ([]string, error) {
	flLock := sync.Mutex{}
	fishList := make([]string, 0)

	c := colly.NewCollector(colly.Async())
	c.OnHTML("div.tb-grid-column", func(e *colly.HTMLElement) {
		h2 := e.DOM.Find("h2.tb-heading")

		flLock.Lock()
		fishList = append(fishList, h2.Text())
		flLock.Unlock()
	})

	err := c.Visit(source)
	if err != nil {
		return nil, err
	}

	c.Wait()
	return fishList, nil
}
