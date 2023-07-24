npx tailwindcss -i web/style.css -o dist/style.css
gofmt -w *.go
go build .
cp -r web/scripts dist/scripts
