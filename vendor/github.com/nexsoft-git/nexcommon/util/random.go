package util

import (
	"math/rand"
	"time"
)

type RandomGatchaWithoutProbabilityResult struct {
	ID      int64                                `json:"id"`
	Name    string                               `json:"name"`
	Results []RandomItemGatchaWithoutProbability `json:"results"`
}

type RandomGatchaWithoutProbability struct {
	ID                  int64                                `json:"id"`
	Name                string                               `json:"name"`
	SpinCount           int                                  `json:"spin_count"`
	RemoveItemAfterSpin bool                                 `json:"is_remove_item"`
	Items               []RandomItemGatchaWithoutProbability `json:"items"`
}

func (r RandomGatchaWithoutProbability) Spin() (
	RandomGatchaWithoutProbabilityResult,
	[]RandomItemGatchaWithoutProbability,
) {

	items := r.Items
	rand.Seed(time.Now().UnixNano())
	var result RandomGatchaWithoutProbabilityResult
	result.ID = r.ID
	result.Name = r.Name

	for i := 0; i < r.SpinCount; i++ {
		min := 0

		if len(items) == 0 {
			continue
		}

		max := len(items) - 1
		spinResult := rand.Intn(max-min+1) + min
		result.Results = append(result.Results, items[spinResult])

		if r.RemoveItemAfterSpin {
			items = append(items[:spinResult], items[spinResult+1:]...)
		}

	}

	return result, items
}

type RandomItemGatchaWithoutProbability struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type RandomItem struct {
	Name        string
	Probability float64
}

func GetRandomValueWithWeight(
	items []RandomItem,
) RandomItem {
	return drawRandomItem(items)

}

func drawRandomItem(items []RandomItem) RandomItem {
	rand.Seed(time.Now().UnixNano())
	totalProbability := 0.0
	for _, item := range items {
		totalProbability += item.Probability
	}

	randomNumber := rand.Float64() * totalProbability

	runningProbability := 0.0
	for _, item := range items {
		runningProbability += item.Probability
		if randomNumber <= runningProbability {
			return item
		}
	}

	return items[len(items)-1]
}
