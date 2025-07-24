cd frontend
npx prettier --write src/
npx eslint
npm run build
cd ..
gofmt -w {cmd,internal}/**/*.go
rm ftag
CGO_ENABLED=0 go build ./cmd/ftag/
