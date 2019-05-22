export GIT_COMMIT=$(git rev-parse --short HEAD)
echo $GIT_COMMIT > winnodeman.version
go get ./...
GOOS=windows GOARCH=386 go build -o "winnodeman.exe" -ldflags "-X main.builddate=`date -u +.%Y%m%d.%H%M%S` -X main.gitversion=$GIT_COMMIT"  ./winnodeman
cp winnodeman.exe ../wcontent/content
cp winnodeman.version ../wcontent/content
git log -1 --pretty="%cd %h %ae %s" >> ../wcontent/updates.md
(cd ../wcontent;git add content/winnodeman.exe;git add content/winnodeman.version;git commit -a -m "Winnodeman.exe version updated";git push origin master)


