npx @tailwindcss/cli -i web/style.css -o dist/style.css
gofmt -w *.go
rm ftag
CGO_ENABLED=0 go build .
cp -r web/scripts dist/scripts
