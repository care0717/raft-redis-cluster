SHELL := /bin/bash
.PHONY: run prepare clean runA runB runC runD runE
#runA runB runC を同時に実⾏する 
run: prepare runA runB runC runD runE
prepare:
	@echo "Preparing..."
	rm -rf /tmp/my-raft-cluster/
	mkdir -p /tmp/my-raft-cluster/{nodeA,nodeB,nodeC,nodeD,nodeE}
	@echo "Prepared"
runA:
	go run main.go --server_id=nodeA --address=localhost:50051 --redis_address=localhost:63791 --data_dir /tmp/my-raft-cluster/nodeA --initial_peers "nodeB=localhost:50052|localhost:63792,nodeC=localhost:50053|localhost:63793,nodeD=localhost:50054|localhost:63794,nodeE=localhost:50055|localhost:63795"
runB:
	go run main.go --server_id=nodeB --address=localhost:50052 --redis_address=localhost:63792 --data_dir /tmp/my-raft-cluster/nodeB --initial_peers "nodeA=localhost:50051|localhost:63791,nodeC=localhost:50053|localhost:63793,nodeD=localhost:50054|localhost:63794,nodeE=localhost:50055|localhost:63795"
runC:
	go run main.go --server_id=nodeC --address=localhost:50053 --redis_address=localhost:63793 --data_dir /tmp/my-raft-cluster/nodeC --initial_peers "nodeA=localhost:50051|localhost:63791,nodeB=localhost:50052|localhost:63792,nodeD=localhost:50054|localhost:63794,nodeE=localhost:50055|localhost:63795"
runD:
	go run main.go --server_id=nodeD --address=localhost:50054 --redis_address=localhost:63794 --data_dir /tmp/my-raft-cluster/nodeD --initial_peers "nodeA=localhost:50051|localhost:63791,nodeB=localhost:50052|localhost:63792,nodeC=localhost:50053|localhost:63793,nodeE=localhost:50055|localhost:63795"
runE:
	go run main.go --server_id=nodeE --address=localhost:50055 --redis_address=localhost:63795 --data_dir /tmp/my-raft-cluster/nodeE --initial_peers "nodeA=localhost:50051|localhost:63791,nodeB=localhost:50052|localhost:63792,nodeC=localhost:50053|localhost:63793,nodeD=localhost:50054|localhost:63794"