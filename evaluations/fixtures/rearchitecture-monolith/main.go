package main

// Server accepts an order, applies business rules, and writes it to storage.
type Server struct {
	Store Store
}

type Store interface {
	Save(order Order) error
}

type Order struct {
	CustomerID string
}
