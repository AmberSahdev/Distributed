all:
	@echo "building all"
	make branch
	make client

branch:
	@echo "building branch"
	go build -o branch ./src/branch/main.go ./src/branch/utils.go

client:
	@echo "building client"
	go build -o client ./src/client/main.go ./src/client/utils.go ./src/client/myTypes.go

launcher:
	@echo "building launcher"
	go build -o launcher launcher.go

clean:
	@echo "deleting binaries"
	rm client branch
