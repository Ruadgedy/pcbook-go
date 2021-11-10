package service

import "sync"

// Rating contains the rating information of a laptop
type Rating struct {
	Count uint32
	Sum float64
}

// RateStore is an interface to store laptop ratings
type RatingStore interface {
	// Add adds a new laptop score to the store and returns its rating
	Add(laptopID string, score float64) (*Rating, error)
}

// InMemoryRatingStore stores laptop ratings in memory
type InMemoryRatingStore struct{
	mutex sync.Mutex
	rating map[string]*Rating
}

func (store *InMemoryRatingStore) Add(laptopID string, score float64) (*Rating, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	rating := store.rating[laptopID]
	if rating == nil {
		rating = &Rating{
			Count: 1,
			Sum:   score,
		}
	}else {
		rating.Count++
		rating.Sum += score
	}

	// TODO: 这里是指针变量，需不需要在重新指回？
	store.rating[laptopID] = rating
	return rating, nil
}

func NewInMemoryRatingStore() *InMemoryRatingStore {
	return &InMemoryRatingStore{
		rating: make(map[string]*Rating),
	}
}


