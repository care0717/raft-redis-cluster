package raft

import (
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/bootjp/go-kvlib/store"
	"github.com/hashicorp/raft"
)

var _ raft.FSM = (*StateMachine)(nil)

type Op int

const (
	Put Op = iota
	Del
)

type KVCmd struct {
	Op    Op     `json:"op"`
	Key   []byte `json:"key"`
	Value []byte `json:"val"`
}

type StateMachine struct {
	store store.Store
}

func NewStateMachine(store store.Store) *StateMachine {
	return &StateMachine{store: store}
}

func (s *StateMachine) Apply(log *raft.Log) interface{} {
	ctx := context.Background()
	c := KVCmd{}

	err := json.Unmarshal(log.Data, &c)
	if err != nil {
		return err
	}
	return s.handleRequest(ctx, c)
}

func (s *StateMachine) Restore(rc io.ReadCloser) error {
	return s.store.Restore(rc)
}

func (s *StateMachine) Snapshot() (raft.FSMSnapshot, error) {
	rc, err := s.store.Snapshot()
	if err != nil {
		return nil, err
	}
	return &KVSnapshot{rc}, nil
}

var ErrUnknownOp = errors.New("unknown operation")

func (s *StateMachine) handleRequest(ctx context.Context, c KVCmd) error {
	switch c.Op {
	case Put:
		return s.store.Put(ctx, c.Key, c.Value)
	case Del:
		return s.store.Delete(ctx, c.Key)
	default:
		return ErrUnknownOp
	}
}
