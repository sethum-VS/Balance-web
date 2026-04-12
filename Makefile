clean:
	go clean
	rm -r -f ./bin

esbuild:
	go run github.com/evanw/esbuild/cmd/esbuild@latest web/src/ts/main.ts --bundle --outfile=web/static/js/main.js

tailwind:
	npx tailwindcss -i web/src/css/input.css -o web/static/css/styles.css --minify

compile:
	@make clean
	@go run github.com/a-h/templ/cmd/templ@latest generate
	mkdir -p ./bin
	go build -o ./bin/server ./cmd/server

compile_linux:
	@make clean
	@go run github.com/a-h/templ/cmd/templ@latest generate
	mkdir -p ./bin
	GOARCH=amd64 GOOS=linux go build -o ./bin/server ./cmd/server

run:
	@make esbuild
	@make tailwind
	@go run github.com/a-h/templ/cmd/templ@latest generate
	@make compile
	./bin/server

install_deps:
	@go install github.com/bokwoon95/wgo@latest
	@go install github.com/a-h/templ/cmd/templ@latest

dev:
	@wgo -file=.go -file=.templ -file=.js -file=.ts -file=.css -xdir=web/static -xfile=_templ.go templ generate :: make esbuild :: make tailwind :: go run ./cmd/server
