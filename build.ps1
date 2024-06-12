$env:GOOS="windows" 
$env:GOARCH="amd64" 
go build -o "./dist/cmd.exe"

# $env:GOOS="linux" 
# $env:GOARCH="amd64" 
# go build -o "../dist/cmd"

# $env:GOOS="darwin" 
# $env:GOARCH="amd64" 
# go build -o "../dist/cmd"
