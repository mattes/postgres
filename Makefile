update-naughty-strings:
	curl -L https://raw.githubusercontent.com/minimaxir/big-list-of-naughty-strings/master/blns.base64.json -o test_data/naughty.b64.json
	@echo "TODO Remove first empty line in test_data/naughty.b64.json"

benchmark:
	go test -mod=vendor -benchmem -bench .

test:
	go test -mod=vendor -v

race:
	go test -mod=vendor -v -race

coverage:
	go test -mod=vendor -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out

