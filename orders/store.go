package main

import "context"

type store struct {
	// add here our mongoDB
}

func NewStore() *store {
	return &store{}
}

func (s *store) Create(context.Context) error {
	return nil
}