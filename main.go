package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/care0717/raft-redis-cluster/raft"
	"github.com/care0717/raft-redis-cluster/store"
	"github.com/care0717/raft-redis-cluster/transport"
	hraft "github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

type initialPeersList []Peer

type Peer struct {
	NodeID    string
	RaftAddr  string
	RedisAddr string
}

func (i *initialPeersList) Set(value string) error {
	node := strings.Split(value, ",")
	for _, n := range node {
		nodes := strings.Split(n, "=")
		if len(nodes) != 2 {
			return fmt.Errorf("invalid peer format: %s, expected NodeID=RaftAddr|RedisAddr", n)
		}

		address := strings.Split(nodes[1], "|")
		if len(address) != 2 {
			return fmt.Errorf("invalid peer format: %s, expected RaftAddr|RedisAddr", n)
		}

		*i = append(*i, Peer{
			NodeID:    nodes[0],
			RaftAddr:  address[0],
			RedisAddr: address[1],
		})
	}
	return nil
}

func (i *initialPeersList) String() string {
	return fmt.Sprintf("%v", *i)
}

var (
	raftAddr     = flag.String("address", "localhost:50051", "TCP host+port for this raft node")
	redisAddr    = flag.String("redis_address", "localhost:6379", "TCP host+port for redis")
	serverID     = flag.String("server_id", "", "unique node ID")
	dataDir      = flag.String("data_dir", "", "path to a data directory for this node")
	initialPeers = initialPeersList{}
)

func init() {
	flag.Var(&initialPeers, "initial_peers", "initial raft peers (NodeID=RaftAddr|RedisAddr)")
	flag.Parse()
	validateFlags()
}

func validateFlags() {
	if *serverID == "" {
		log.Fatal("server_id is required")
	}
	if *raftAddr == "" {
		log.Fatal("raft_address is required")
	}
	if *redisAddr == "" {
		log.Fatal("redis_address is required")
	}
	if *dataDir == "" {
		log.Fatal("data_dir is required")
	}
}

func main() {
	datastore := store.NewMemoryStore()
	st := raft.NewStateMachine(datastore)
	r, sdb, err := NewRaft(*dataDir, *serverID, *raftAddr, st, initialPeers)
	if err != nil {
		log.Fatalln(err)
	}

	redis := transport.NewRedis(hraft.ServerID(*serverID), r, datastore, sdb)
	err = redis.Serve(*redisAddr)
	if err != nil {
		log.Fatalln(err)
	}
}

const snapshotRetainCount = 2

func NewRaft(baseDir, id, address string, fsm hraft.FSM, nodes initialPeersList) (*hraft.Raft, hraft.StableStore, error) {
	c := hraft.DefaultConfig()
	c.LocalID = hraft.ServerID(id)

	ldb, err := raftboltdb.NewBoltStore(filepath.Join(baseDir, "logs.dat"))
	if err != nil {
		return nil, nil, err
	}

	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(baseDir, "stable.dat"))
	if err != nil {
		return nil, nil, err
	}

	fss, err := hraft.NewFileSnapshotStore(baseDir, snapshotRetainCount, os.Stderr)
	if err != nil {
		return nil, nil, err
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, nil, err
	}

	tm, err := hraft.NewTCPTransport(address, tcpAddr, 10, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, nil, err
	}

	r, err := hraft.NewRaft(c, fsm, ldb, stableStore, fss, tm)
	if err != nil {
		return nil, nil, err
	}

	cfg := hraft.Configuration{
		Servers: []hraft.Server{
			{
				Suffrage: hraft.Voter,
				ID:       hraft.ServerID(id),
				Address:  hraft.ServerAddress(address),
			},
		},
	}

	for _, peer := range nodes {
		sid := hraft.ServerID(peer.NodeID)
		cfg.Servers = append(cfg.Servers, hraft.Server{
			Suffrage: hraft.Voter,
			ID:       sid,
			Address:  hraft.ServerAddress(peer.RaftAddr),
		})

		err := store.SetRedisAddrByNodeID(stableStore, sid, peer.RedisAddr)
		if err != nil {
			return nil, nil, err
		}
	}

	f := r.BootstrapCluster(cfg)
	if err := f.Error(); err != nil {
		return nil, nil, err
	}

	return r, stableStore, nil

}
