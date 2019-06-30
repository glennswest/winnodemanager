export GIT_COMMIT=$(git rev-parse --short HEAD)
echo $GIT_COMMIT > winnodeman.version
export wcontent=winoperatordata
go get ./...
GOOS=windows GOARCH=386 go build -o "winnodeman.exe" -ldflags "-X main.builddate=`date -u +.%Y%m%d.%H%M%S` -X main.gitversion=$GIT_COMMIT"  ./winnodeman
cp winnodeman.exe ../${wcontent}/wcontent/content
cp winnodeman.version ../${wcontent}/wcontent/content
export GITREASON=`git log -1 --pretty="winnodeman: %cd %h %ae %s"`
echo ${GITREASON} >> ../${wcontent}/updates.md
(cd ../${wcontent};git add wcontent/content/winnodeman.exe;git add wcontent/content/winnodeman.version;git commit  -a -m "Update winnodeman executable";git push origin master)


