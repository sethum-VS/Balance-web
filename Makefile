clean:
	go clean
	rm -r -f ./bin

compile:
	@make clean
	@templ generate
	mkdir -p ./bin
	go build -o ./bin/server ./cmd/server

compile_linux:
	@make clean
	@templ generate
	mkdir -p ./bin
	GOARCH=amd64 GOOS=linux go build -o ./bin/server ./cmd/server

run:
	@npm run tailwind
	@npm run esbuild
	@templ generate
	@make compile
	./bin/server

install_deps:
	@go install github.com/bokwoon95/wgo@latest
	@go install github.com/a-h/templ/cmd/templ@latest
	@npm install

dev:
	@wgo -file=.go -file=.templ -file=.js -file=.ts -xdir=node_modules -xdir=web/static -xfile=_templ.go templ generate :: npm run tailwind :: npm run esbuild :: go run ./cmd/server
