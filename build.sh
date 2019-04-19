export GIT_COMMIT=$(git rev-parse --short HEAD)
echo $GIT_COMMIT > winnodeman.version
go get ./...
GOOS=windows GOARCH=386 go build -o "winnodeman.exe" -ldflags "-X main.builddate=`date -u +.%Y%m%d.%H%M%S` -X main.gitversion=$GIT_COMMIT"  ./winnodeman

