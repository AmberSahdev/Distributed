all: build_logger build_node

build_logger:
	@echo "building logger"
	go build logger.go

build_node:
	@echo "building node"
	go build node.go

clean:
	@echo "deleting binaries"
	rm node && rm logger