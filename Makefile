clean:
	go clean
	rm -r -f ./bin

esbuild:
	go run github.com/evanw/esbuild/cmd/esbuild@latest web/src/ts/main.ts --bundle --outfile=web/static/js/main.js

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
	@go run github.com/a-h/templ/cmd/templ@latest generate
	@make compile
	./bin/server

install_deps:
	@go install github.com/bokwoon95/wgo@latest
	@go install github.com/a-h/templ/cmd/templ@latest

dev:
	@wgo -file=.go -file=.templ -file=.js -file=.ts -xdir=web/static -xfile=_templ.go templ generate :: make esbuild :: go run ./cmd/server
