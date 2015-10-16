Write-Host "Building Depro for Windows"
$GIT_DESCRIBE=git describe --tags
$GIT_COMMIT=git rev-parse HEAD
go build -ldflags "-X main.GitDescribe $GIT_DESCRIBE -X main.GitCommit $GIT_COMMIT" -o bin/depro.exe
