.PHONY: templ css run dev

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
