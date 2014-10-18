GOROOT=/d/dev/go/go
GOPATH=/d/dev/go/mypath
PKG=github.com/noypi/ngweb

DATE=ngweb-$(date +%Y%m%d)

echo building windows/386...
GOOS=windows CGO_ENABLED=0 GOARCH=386   go build -o ngweb_win32.exe $PKG

echo building windows/64
GOOS=windows CGO_ENABLED=0 GOARCH=amd64 go build -o ngweb_win64.exe $PKG

echo building linux/64
GOOS=linux CGO_ENABLED=1 GOARCH=amd64   go build -o ngweb_linux64 $PKG

echo compressing files...
zip ngweb_win32-$DATE.zip ngweb_win32.exe
zip ngweb_win64-$DATE.zip ngweb_win64.exe
gzip -c ngweb_linux64 > ngweb_linux64-$DATE.gz

rm ngweb_win*.exe ngweb_linux64

cd ..

