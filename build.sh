npx tailwindcss -i web/style.css -o dist/style.css
gofmt -w *.go
CGO_ENABLED=0 go build .
cp -r web/scripts dist/scripts
