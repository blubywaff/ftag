npx tailwindcss@v3.4 -i web/style.css -o dist/style.css
gofmt -w {cmd,internal}/**/*.go
rm ftag
CGO_ENABLED=0 go build ./cmd/ftag/
cp -r web/scripts dist/scripts
