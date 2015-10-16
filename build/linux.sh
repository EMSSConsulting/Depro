echo "Building Depro for Linux"
go build -ldflags "-X main.GitDescribe=$(git describe --tags) -X main.GitCommit=$(git rev-parse HEAD)" -o bin/depro
