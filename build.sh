export GIT_COMMIT=$(git rev-parse --short HEAD)
echo $GIT_COMMIT > winnodeman.version
go get ./...
GOOS=windows GOARCH=386 go build -o "winnodeman.exe" -ldflags "-X main.builddate=`date -u +.%Y%m%d.%H%M%S` -X main.gitversion=$GIT_COMMIT" 
cp winnodeman/winnodeman.exe ../winoperatordata/wcontent/content
cp winnodeman.version ../winoperatordata/wcontent/content
export GITREASON=`git log -1 --pretty="winnodeman: %cd %h %ae %s"`
echo ${GITREASON} >> ../winoperatordata/updates.md
(cd ../winoperatordata;git add -f wcontent/content/winnodeman.exe;git add -f wcontent/content/winnodeman.version;git add updates.md;git commit -a -m "Update winnodeman executable";git push origin master)


