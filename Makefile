.PHONY: templ css run dev test test-unit test-integration test-coverage test-e2e test-all

templ:
	templ generate

css:
	npx @tailwindcss/cli -i input.css -o static/css/output.css

run: templ css
	go run main.go serve

dev:
	templ generate --watch &
	npx @tailwindcss/cli -i input.css -o static/css/output.css --watch &
	go run main.go serve

test:
	go test ./... -count=1

test-unit:
	go test ./services/... -v -count=1

test-integration:
	go test ./handlers/... -v -count=1

test-coverage:
	go test ./... -coverprofile=coverage.out -count=1
	go tool cover -func=coverage.out | grep total

test-e2e:
	cd tests/e2e && npm install && npx playwright install chromium && npx playwright test

test-all: test test-e2e
